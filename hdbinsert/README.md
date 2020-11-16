# hdbinsert

hdbinsert was created in the context of a performance / throughput analysis of different hdb client implementations.

## Test object

Test object is a column table consisting of 10 columns, one of type integer and 9 of type double:

```
create column table GOMESSAGE (DEVICEID INTEGER, TEMPERATUR DOUBLE, HUMIDITY DOUBLE, CO2 DOUBLE, CO DOUBLE, LPG DOUBLE, SMOKE DOUBLE, PRESENCE DOUBLE, LIGHT DOUBLE, SOUND DOUBLE)
```

Whereas the integer column is used like a counter the double columns are filled randomly withing a fixed (hard-coded) range.
Anyway, as long as the content of the columns is not NULL,
**the column value does not have any performance impact, as the hdb protocol is using a fixed size exchange format for these data types**.

## Test variants

The basic idea is to insert data in chunks (batchCount) of a fixed amount of records (batchSize) whether sequentially or 'in parallel'.
The actual 'grade of parallelization' is heavily depending on the test environment (CPU cores, TCP/IP stack). hdbinsert 'enables'
potential parallelism 'idiomatically' via Goroutines. Each Goroutine is using an own dedicated database connection for the tests
being independent of the Go sql.DB connection pool handling and configuration.
As the test performance results are heavily 'I/O bound' the implementation mainly tries to reduce client server round-trips. Therefore
the go-hdb driver capabilities 'bulk' and 'many' are used (please refer to the [go-hdb driver documentation and examples](https://github.com/SAP/go-hdb)
for details).

Difference of 'bulk' and 'many' in a nutshell:
* 'bulk'
	* records are inserted individually via stmt.Exec(column, ...), stored in an internal go-hdb buffer and 'flushed' explicitly via stmt.Exec()
	* pro: less memory consumption as record collection does not need to be build in application memory
	* con: stmt.Exec(column, ...) overhead for each call
* 'many'
	* records are inserted as collection and 'flushed' automatically be the go-hdb driver
	* pro: less overhead for stmt.Exec(column, ...) calls
	* con: higher memory consumption as record collection need to be build in application memory before call
	
## In a real world example...

... one might consider

* to implement a worker pool with the number of concurrent workers set in relation to GOMAXPROCS
* optimizing the number of records per chunk (batchSize)
	* the hdb protocol does allow max. 32767 records per message
	* hdbinsert sets the BulkSize to batchSize via the driver.Connector object but the max. BulkSize is equal to the max. number of records allowed per message
	* so, when reaching the max. number of records per message, go-hdb 'flushes' the data 'under the hood' which is triggering a client server round-trip
* optimizing the go-hdb driver TCP/IP buffer size.
	* all writes to the TCP/IP connection are buffered by the go-hdb client
	* the buffer size can be configured via the driver.Connector object (BufferSize)
	* when reaching the buffer size, the go-hdb driver writes the buffered data to the TCP/IP connection

## Execute tests

**Caution: please do NOT use a productive HANA instance for testing as hdbinsert does allow to modify and even drop schemas and / or database tables.**

Executing hdbinsert starts a HTTP server on 'localhost:8080'.

After starting a browser pointing to the server address the following HTML page should be visible in the browser window:

![cannot display hdbinsert.png](./hdbinsert.png)
 
* the first section displays some runtime information like GOMAXPROCS and the driver and database version
* the second section lists all test relevant parameters which can be set as environment variables or commandline parameters starting hdbinsert
* the third sections allows to execute tests with predefined BatchCount and BatchSize parameters (see parameters command-line flag)
* the last section provides some database operations for the selected test database schema and table

Clicking on one of the predefined test will execute it and display the result consisting of test parameters and the 'insert' duration in seconds.
The result is a JSON payload, which provides an easy way to be interpreted by a program.

## URL format 

Running hdbinsert as HTTP server a test can be executed via a HTTP GET using the following URL format:

```
http://<host>:<port>/test/<TestType>?batchcount=<number>&batchsize=<number>
```
with 
```
<TestType> =:= BulkSeq | ManySeq | BulkPar | ManyPar
```

## Benchmark

Parallel to the single execution using the browser or any other HTTP client (like wget, curl, ...), the tests can be executed automatically
as Go benchmark. The benchmark can be found and executed in the [benchmark subdirectory](./benchmark) whether by 
```
go test -bench .
```
or compiling the benchmark with 
```
go test -c 
```
and executing it via
```
./benchmark.test -test.bench .
```

The benchmark is 'self-contained', meaning it includes its own http server (for details please see [httptest](https://golang.org/pkg/net/http/httptest/).

In addition to the standard Go benchmarks four additional metrics are reported:
* avgsec/op: the average time (*) 
* maxsec/op: the maximum time (*)
* medsec/op: the median  time (*)
* minsec/op: the minimal time (*)

(*) inserting BatchCount x BatchSize records into the database table when executing one test several times.

For details about Go benchmarks please see the [Golang testing documentation](https://golang.org/pkg/testing).

### Benchmark examples

Finally let's see some examples executing the benchmark.

```
export GOHDBDSN="hdb://MyUser:MyPassword@host:port"
cd benchmark
go test -c 
```

* set the data source name (dsn) via environment variable
* change to benchmark directory
* and compile the benchmark


```
./benchmark.test -test.bench . -test.benchtime 10x
```

* -test.bench . (run all benchmarks)
* -test.benchtime 10x (run each benchmark ten times)
* run benchmarks for all BatchCount / BatchSize combinations defined as parameters 
* the test database table is dropped and re-created before each benchmark execution (command-line parameter drop defaults to true)

```
./benchmark.test -test.bench . -test.benchtime 10x -parameters "10x10000"
```
* same like before but
* execute benchmarks only for 10x10000 as BatchCount / BatchSize combination

```
./benchmark.test -test.bench . -test.benchtime 10x -wait 5 
```

* same like first example and
* -wait 5 (wait 5 seconds before starting a benchmark run to reduce database pressure)

```
./benchmark.test -test.bench . -test.benchtime 10x -wait 5 -separate
```

* same like before and
* -separate (create own table for the 'parallel' benchmarks - table name: \<tablename\>\_\<number\> with 0 <= \<number\> < batchCount)

### Benchmark example output

```
./benchmark.test -test.bench . -test.benchtime 10x -wait 5

GOMAXPROCS: 32
NumCPU: 32
Driver Version: 0.102.3
HANA Version: 2.00.045.00.1575639312
goos: linux
goarch: amd64
pkg: github.com/stfnmllr/go-hdb-test/hdbinsert/benchmark
BenchmarkInsert/1x100000/bulkSeq-32 	      10	5491625426 ns/op	         0.386 avgsec/op	         0.401 maxsec/op	         0.386 medsec/op	         0.372 minsec/op
BenchmarkInsert/1x100000/manySeq-32 	      10	5411059768 ns/op	         0.306 avgsec/op	         0.334 maxsec/op	         0.303 medsec/op	         0.277 minsec/op
BenchmarkInsert/1x100000/bulkPar-32 	      10	5467641640 ns/op	         0.360 avgsec/op	         0.381 maxsec/op	         0.359 medsec/op	         0.346 minsec/op
BenchmarkInsert/1x100000/manyPar-32 	      10	5409775781 ns/op	         0.306 avgsec/op	         0.333 maxsec/op	         0.304 medsec/op	         0.289 minsec/op
BenchmarkInsert/10x10000/bulkSeq-32 	      10	5589876312 ns/op	         0.475 avgsec/op	         0.517 maxsec/op	         0.466 medsec/op	         0.443 minsec/op
BenchmarkInsert/10x10000/manySeq-32 	      10	5468570802 ns/op	         0.363 avgsec/op	         0.378 maxsec/op	         0.359 medsec/op	         0.352 minsec/op
BenchmarkInsert/10x10000/bulkPar-32 	      10	5497928724 ns/op	         0.199 avgsec/op	         0.224 maxsec/op	         0.197 medsec/op	         0.184 minsec/op
BenchmarkInsert/10x10000/manyPar-32 	      10	5473941266 ns/op	         0.187 avgsec/op	         0.195 maxsec/op	         0.190 medsec/op	         0.175 minsec/op
BenchmarkInsert/100x1000/bulkSeq-32 	      10	5891522728 ns/op	         0.771 avgsec/op	         0.803 maxsec/op	         0.768 medsec/op	         0.753 minsec/op
BenchmarkInsert/100x1000/manySeq-32 	      10	5763993305 ns/op	         0.657 avgsec/op	         0.675 maxsec/op	         0.654 medsec/op	         0.645 minsec/op
BenchmarkInsert/100x1000/bulkPar-32 	      10	7146420424 ns/op	         0.299 avgsec/op	         0.323 maxsec/op	         0.301 medsec/op	         0.277 minsec/op
BenchmarkInsert/100x1000/manyPar-32 	      10	7119091846 ns/op	         0.309 avgsec/op	         0.332 maxsec/op	         0.309 medsec/op	         0.289 minsec/op
BenchmarkInsert/1x1000000/bulkSeq-32         	      10	10156033702 ns/op	     4.47 avgsec/op	         4.57 maxsec/op	         4.46 medsec/op	         4.40 minsec/op
BenchmarkInsert/1x1000000/manySeq-32         	      10	9013212595 ns/op	         3.50 avgsec/op	         3.65 maxsec/op	         3.49 medsec/op	         3.30 minsec/op
BenchmarkInsert/1x1000000/bulkPar-32         	      10	9564600274 ns/op	         4.05 avgsec/op	         4.14 maxsec/op	         4.07 medsec/op	         3.91 minsec/op
BenchmarkInsert/1x1000000/manyPar-32         	      10	9046538676 ns/op	         3.55 avgsec/op	         3.59 maxsec/op	         3.56 medsec/op	         3.52 minsec/op
BenchmarkInsert/10x100000/bulkSeq-32         	      10	10222504416 ns/op	     4.53 avgsec/op	         4.74 maxsec/op	         4.55 medsec/op	         4.23 minsec/op
BenchmarkInsert/10x100000/manySeq-32         	      10	9236246972 ns/op	         3.61 avgsec/op	         3.72 maxsec/op	         3.61 medsec/op	         3.52 minsec/op
BenchmarkInsert/10x100000/bulkPar-32         	      10	7733685534 ns/op	         1.93 avgsec/op	         1.99 maxsec/op	         1.92 medsec/op	         1.86 minsec/op
BenchmarkInsert/10x100000/manyPar-32         	      10	7668492132 ns/op	         1.92 avgsec/op	         2.13 maxsec/op	         1.90 medsec/op	         1.78 minsec/op
BenchmarkInsert/100x10000/bulkSeq-32         	      10	11273520788 ns/op	     5.47 avgsec/op	         5.74 maxsec/op	         5.45 medsec/op	         5.34 minsec/op
BenchmarkInsert/100x10000/manySeq-32         	      10	10055706775 ns/op	     4.34 avgsec/op	         4.46 maxsec/op	         4.34 medsec/op	         4.25 minsec/op
BenchmarkInsert/100x10000/bulkPar-32         	      10	10156310805 ns/op	     2.38 avgsec/op	         2.70 maxsec/op	         2.37 medsec/op	         2.25 minsec/op
BenchmarkInsert/100x10000/manyPar-32         	      10	10069910603 ns/op	     2.37 avgsec/op	         2.45 maxsec/op	         2.39 medsec/op	         2.24 minsec/op
BenchmarkInsert/1000x1000/bulkSeq-32         	      10	15017978929 ns/op	     9.14 avgsec/op	         9.24 maxsec/op	         9.12 medsec/op	         9.00 minsec/op
BenchmarkInsert/1000x1000/manySeq-32         	      10	13740478158 ns/op	     7.97 avgsec/op	         8.05 maxsec/op	         7.97 medsec/op	         7.91 minsec/op
BenchmarkInsert/1000x1000/bulkPar-32         	      10	27074190075 ns/op	     3.90 avgsec/op	         4.03 maxsec/op	         3.89 medsec/op	         3.80 minsec/op
BenchmarkInsert/1000x1000/manyPar-32         	      10	26558184657 ns/op	     3.90 avgsec/op	         4.13 maxsec/op	         3.89 medsec/op	         3.79 minsec/op
PASS
```