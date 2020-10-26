// SPDX-FileCopyrightText: 2020 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/SAP/go-hdb/driver"
	"github.com/stfnmllr/go-hdb-test/hdbinsert/env"
)

// Test URL paths.
const (
	TestBulkSeq = "/test/BulkSeq"
	TestManySeq = "/test/ManySeq"
	TestBulkPar = "/test/BulkPar"
	TestManyPar = "/test/ManyPar"
)

// TestResult is the structure used to provide the JSON based test result response.
type TestResult struct {
	Test       string
	Seconds    float64
	BatchCount int
	BatchSize  int
	BulkSize   int
	Duration   time.Duration
	Error      string
}

func (r *TestResult) String() string {
	if r.Error != "" {
		return r.Error
	}
	return fmt.Sprintf("%s: insert of %d rows in %f seconds (batchCount %d batchSize %d bulkSize %d)", r.Test, r.BatchCount*r.BatchSize, r.Duration.Seconds(), r.BatchCount, r.BatchSize, r.BulkSize)
}

type testFunc func(db *sql.DB, batchCount, batchSize int) (time.Duration, error)

// TestHandler implements the http.Handler interface for the tests.
type TestHandler struct {
	log        logFunc
	dsn        string
	schemaName string
	tableName  driver.Identifier
	testFuncs  map[string]testFunc

	bulkInsertQuery, insertQuery string
}

// NewTestHandler returns a new TestHandler instance.
func NewTestHandler(log logFunc) (*TestHandler, error) {
	tableName := driver.Identifier(env.TableName())

	h := &TestHandler{log: log, dsn: env.DSN(), schemaName: env.SchemaName(), tableName: tableName}
	h.testFuncs = map[string]testFunc{
		TestBulkSeq: h.bulkSeq,
		TestManySeq: h.manySeq,
		TestBulkPar: h.bulkPar,
		TestManyPar: h.manyPar,
	}
	h.bulkInsertQuery = fmt.Sprintf("bulk insert into %s values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", tableName)
	h.insertQuery = fmt.Sprintf("insert into %s values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", tableName)
	return h, nil
}

func (h *TestHandler) tests() []string {
	// need correct sort order
	return []string{TestBulkSeq, TestManySeq, TestBulkPar, TestManyPar}
}

func (h *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to get a comparable environment for each run
	// by clearing garbage from previous runs.
	runtime.GC()

	batchCount := h.convert2int(r.URL.Query().Get("batchcount"), env.BatchCount())
	batchSize := h.convert2int(r.URL.Query().Get("batchsize"), env.BatchSize())

	test := r.URL.Path

	result := &TestResult{Test: test, BatchCount: batchCount, BatchSize: batchSize}

	defer func() {
		h.log("%s", result)
		e := json.NewEncoder(w)
		e.Encode(result) // ignore error
	}()

	db, bulkSize, err := h.setup(batchSize)
	if err != nil {
		result.Error = err.Error()
		return
	}
	defer h.teardown(db)

	var d time.Duration

	if f, ok := h.testFuncs[test]; ok {
		d, err = f(db, batchCount, batchSize)
	} else {
		err = fmt.Errorf("Invalid test %s", test)
	}

	result.BulkSize = bulkSize
	result.Duration = d
	result.Seconds = d.Seconds()
	if err != nil {
		result.Error = err.Error()
	}
}

func (h *TestHandler) bulkSeq(db *sql.DB, batchCount, batchSize int) (time.Duration, error) {
	numRow := batchCount * batchSize

	conn, err := db.Conn(context.Background())
	if err != nil {
		return 0, err
	}

	stmt, err := conn.PrepareContext(context.Background(), h.bulkInsertQuery)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var d time.Duration

	for i := 0; i < numRow; i++ {
		row := randRow(i)
		t := time.Now()
		if _, err := stmt.Exec(row...); err != nil {
			return d, err
		}
		d += time.Since(t)
	}

	// Call final stmt.Exec().
	t := time.Now()
	if _, err := stmt.Exec(); err != nil {
		return d, err
	}
	d += time.Since(t)

	return d, nil
}

func (h *TestHandler) manySeq(db *sql.DB, batchCount, batchSize int) (time.Duration, error) {
	conn, err := db.Conn(context.Background())
	if err != nil {
		return 0, err
	}

	stmt, err := conn.PrepareContext(context.Background(), h.insertQuery)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var d time.Duration

	for i := 0; i < batchCount; i++ {
		rows := randRows(i, batchSize)
		t := time.Now()
		if _, err := stmt.Exec(rows); err != nil {
			return d, err
		}
		d += time.Since(t)
	}

	return d, nil
}

type task struct {
	conn *sql.Conn
	stmt *sql.Stmt
	rows [][]interface{}
	err  error
}

func newTask(db *sql.DB, query string, i, size int) (*task, error) {
	conn, err := db.Conn(context.Background())
	if err != nil {
		return nil, err
	}

	stmt, err := conn.PrepareContext(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return &task{conn: conn, stmt: stmt, rows: randRows(i, size)}, nil
}

func (t *task) close() {
	t.stmt.Close()
	t.conn.Close()
}

func (h *TestHandler) createTasks(db *sql.DB, query string, batchCount, batchSize int) ([]*task, error) {
	var err error
	tasks := make([]*task, batchCount)
	for i := 0; i < batchCount; i++ {
		if tasks[i], err = newTask(db, query, i, batchSize); err != nil {
			return nil, err
		}
	}
	return tasks, err
}

func (h *TestHandler) bulkPar(db *sql.DB, batchCount, batchSize int) (time.Duration, error) {
	var wg sync.WaitGroup

	tasks, err := h.createTasks(db, h.bulkInsertQuery, batchCount, batchSize)
	if err != nil {
		return 0, err
	}

	t := time.Now() // Start time.

	for i, t := range tasks { // Start one worker per task.
		wg.Add(1)

		go func(worker int, t *task) {
			defer wg.Done()

			for _, row := range t.rows {
				if _, err := t.stmt.Exec(row...); err != nil {
					t.err = err
				}
			}
			// Call final stmt.Exec().
			if _, err := t.stmt.Exec(); err != nil {
				t.err = err
			}

		}(i, t)
	}
	wg.Wait()

	d := time.Since(t) // Duration.

	for _, t := range tasks {
		// return last error
		err = t.err
		t.close()
	}

	return d, err
}

func (h *TestHandler) manyPar(db *sql.DB, batchCount, batchSize int) (time.Duration, error) {
	var wg sync.WaitGroup

	tasks, err := h.createTasks(db, h.insertQuery, batchCount, batchSize)
	if err != nil {
		return 0, err
	}

	t := time.Now() // Start time.

	for i, t := range tasks { // Start one worker per task.
		wg.Add(1)

		go func(worker int, t *task) {
			defer wg.Done()

			if _, err := t.stmt.Exec(t.rows); err != nil {
				t.err = err
			}

		}(i, t)
	}
	wg.Wait()

	d := time.Since(t) // Duration.

	for _, t := range tasks {
		// return last error
		err = t.err
		t.close()
	}

	return d, err
}

func (h *TestHandler) setup(batchSize int) (*sql.DB, int, error) {
	// Set bulk size to batchSize.
	connector, err := driver.NewConnector(map[string]interface{}{"dsn": h.dsn, "defaultSchema": h.schemaName, "bulkSize": batchSize, "bufferSize": env.BufferSize()})
	if err != nil {
		return nil, 0, err
	}
	return sql.OpenDB(connector), connector.BulkSize(), nil
}

func (h *TestHandler) teardown(db *sql.DB) {
	db.Close()
}

// convert2int converts the string s to an integer.
// If s is empty the defaultValue is returned.
// If the conversion is not posible the program will be exited.
func (h *TestHandler) convert2int(s string, defaultValue int) int {
	if s == "" { // empty string -> return default value
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		h.log("Integer conversion %s failed: %s", s, err)
		return defaultValue
	}
	return i
}

// randFloat64 return a random float64 number f with min <= f < max.
func randFloat64(min, max float64) float64 {
	return rand.Float64()*(max-min) + min
}

// randRow returns a table row with random float64 fields.
func randRow(idx int) []interface{} {
	return []interface{}{idx, randFloat64(25, 26), randFloat64(40, 60), randFloat64(500, 600), randFloat64(0.9, 1.1), randFloat64(23, 25), randFloat64(50, 60), randFloat64(0, 1), randFloat64(600, 800), randFloat64(400, 500)}
}

// randRows returns size table rows with random float64 fields.
func randRows(i, size int) [][]interface{} {
	rows := make([][]interface{}, size)
	for j := 0; j < size; j++ {
		rows[j] = randRow(i*size + j)
	}
	return rows
}
