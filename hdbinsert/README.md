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
* the third sections allows to execute tests with predefined BatchCount and BatchSize parameters
* the last seconds provides some database operations for the selected test database schema and table

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

(*) inserting BatchCount * BatchSize records into the database table when executing one test several times.

* for details about Go benchmarks please see the [Golang testing documentation](https://golang.org/pkg/testing)

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
./benchmark.test -test.bench . -test.benchtime 10x -batchSize 10000 -batchCount 10
```

* -test.bench . (run all benchmarks)
* -test.benchtime 10x (run each benchmark ten times)
* -batchSize (write chunks of 10.000 records)
* -batchCount (10 chunks whether sequential or 'parallel')

```
./benchmark.test -test.bench . -test.benchtime 10x -batchSize 10000 -batchCount 10 -drop -wait 5 
```

* same like before and
* -drop (drop and recreate database table before each benchmark execution)
* -wait 5 (wait 5 seconds before starting each benchmark to reduce database pressure)

```
./benchmark.test -test.bench . -test.benchtime 10x -batchSize 10000 -batchCount 10 -drop -wait 5 -separate
```

* same like before and
* -separate (create own table for the 'parallel' benchmarks - table name: <tablename>_<number> with 0 <= <number> < batchCount)

### Benchmark example output

```
./benchmark.test -test.bench . -test.benchtime 10x -batchSize 10000 -batchCount 10 -drop -wait 5 

GOMAXPROCS: 32
NumCPU: 32
Driver Version: 0.102.3
HANA Version: 2.00.045.00.1575639312
goos: linux
goarch: amd64
pkg: github.com/stfnmllr/go-hdb-test/hdbinsert/benchmark
BenchmarkInsert/bulkSeq-32         	      10	5601837358 ns/op	         0.483 avgsec/op	         0.513 maxsec/op	         0.486 medsec/op	         0.446 minsec/op
BenchmarkInsert/manySeq-32         	      10	5474729249 ns/op	         0.370 avgsec/op	         0.393 maxsec/op	         0.367 medsec/op	         0.362 minsec/op
BenchmarkInsert/bulkPar-32         	      10	5500516542 ns/op	         0.202 avgsec/op	         0.214 maxsec/op	         0.203 medsec/op	         0.189 minsec/op
BenchmarkInsert/manyPar-32         	      10	5481515209 ns/op	         0.193 avgsec/op	         0.206 maxsec/op	         0.192 medsec/op	         0.183 minsec/op
PASS
```