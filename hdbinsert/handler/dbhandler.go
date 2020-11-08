// SPDX-FileCopyrightText: 2020 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/SAP/go-hdb/driver"
	"github.com/stfnmllr/go-hdb-test/hdbinsert/env"
)

const columns = "DEVICEID INTEGER, TEMPERATUR DOUBLE, HUMIDITY DOUBLE, CO2 DOUBLE, CO DOUBLE, LPG DOUBLE, SMOKE DOUBLE, PRESENCE DOUBLE, LIGHT DOUBLE, SOUND DOUBLE"

// Database operation URL paths.
const (
	CmdCountRows    = "/db/countRows"
	CmdDeleteRows   = "/db/deleteRows"
	CmdCreateTable  = "/db/createTable"
	CmdDropTable    = "/db/dropTable"
	CmdCreateSchema = "/db/createSchema"
	CmdDropSchema   = "/db/dropSchema"
)

const (
	objTable = iota
	objSchema
)

var dbObjText = map[dbObj]string{objTable: "table", objSchema: "schema"}

type dbObj int

func (o dbObj) String() string { return dbObjText[o] }

const (
	opCountRows = iota
	opDeleteRows
	opCreate
	opDrop
)

var dbOpText = map[dbOp]string{opCountRows: "Count rows", opDeleteRows: "Delete rows", opCreate: "Create", opDrop: "Drop"}

type dbOp int

func (o dbOp) String() string { return dbOpText[o] }

// DBResult is the structure used to provide the JSON based cb command result response.
type DBResult struct {
	Command string
	DbObj   dbObj
	DbOp    dbOp
	ObjName string
	NumRow  int64
	Error   string
}

func (r *DBResult) String() string {
	switch {
	case r.Error != "":
		return fmt.Sprintf("%s %s %s error: %s", r.DbOp, r.DbObj, r.ObjName, r.Error)
	case r.NumRow != -1:
		return fmt.Sprintf("%s %s %s: %d rows", r.DbOp, r.DbObj, r.ObjName, r.NumRow)
	default:
		return fmt.Sprintf("%s %s %s: ok", r.DbOp, r.DbObj, r.ObjName)
	}
}

type dbFunc struct {
	Command string
	Obj     dbObj
	Op      dbOp
	f       func(q *urlQuery, r *DBResult) error
}

// DBHandler implements the http.Handler interface for database operations.
type DBHandler struct {
	log     logFunc
	db      *sql.DB
	columns string
	dbFuncs map[string]*dbFunc
}

// NewDBHandler returns a new DBHandler instance.
func NewDBHandler(log logFunc) (*DBHandler, error) {
	connector, err := driver.NewDSNConnector(env.DSN())
	if err != nil {
		return nil, err
	}
	h := &DBHandler{log: log, db: sql.OpenDB(connector), columns: columns}
	h.dbFuncs = map[string]*dbFunc{
		CmdCountRows:    {Command: CmdCountRows, Obj: objTable, Op: opCountRows, f: h.countRows},
		CmdDeleteRows:   {Command: CmdDeleteRows, Obj: objTable, Op: opDeleteRows, f: h.deleteRows},
		CmdCreateTable:  {Command: CmdCreateTable, Obj: objTable, Op: opCreate, f: h.createTable},
		CmdDropTable:    {Command: CmdDropTable, Obj: objTable, Op: opDrop, f: h.dropTable},
		CmdCreateSchema: {Command: CmdCreateSchema, Obj: objSchema, Op: opCreate, f: h.createSchema},
		CmdDropSchema:   {Command: CmdDropSchema, Obj: objSchema, Op: opDrop, f: h.dropSchema},
	}
	return h, nil
}

// DriverVersion returns the go-hdb driver version.
func (h DBHandler) DriverVersion() string { return driver.DriverVersion }

// HDBVersion returns the hdb version.
func (h DBHandler) HDBVersion() string {
	conn, err := h.db.Conn(context.Background())
	if err != nil {
		return err.Error()
	}
	var hdbVersion string
	conn.Raw(func(driverConn interface{}) error {
		hdbVersion = driverConn.(*driver.Conn).ServerInfo().Version.String()
		return nil
	})
	return hdbVersion
}

func (h DBHandler) schemaFuncs() []*dbFunc {
	return []*dbFunc{
		h.dbFuncs[CmdCreateSchema],
		h.dbFuncs[CmdDropSchema],
	}
}

func (h DBHandler) tableFuncs() []*dbFunc {
	return []*dbFunc{
		h.dbFuncs[CmdCreateTable],
		h.dbFuncs[CmdDropTable],
		h.dbFuncs[CmdDeleteRows],
		h.dbFuncs[CmdCountRows],
	}
}

func (h *DBHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	command := r.URL.Path

	result := &DBResult{Command: command}

	defer func() {
		h.log("%s", result)
		e := json.NewEncoder(w)
		e.Encode(result) // ignore error
	}()

	var err error

	dbFunc, ok := h.dbFuncs[command]
	if ok {
		result.DbObj = dbFunc.Obj
		result.DbOp = dbFunc.Op
		err = dbFunc.f(newURLQuery(r), result)
	} else {
		err = fmt.Errorf("Invalid command %s", command)
	}
	if err != nil {
		result.Error = err.Error()
	}
}

func getSchemaTableNames(q *urlQuery) (string, string, error) {
	schemaName, err := q.get(urlQuerySchemaName)
	if err != nil {
		return "", "", err
	}
	tableName, err := q.get(urlQueryTableName)
	if err != nil {
		return "", "", err
	}
	return schemaName, tableName, nil
}

func (h *DBHandler) countRows(q *urlQuery, r *DBResult) error {
	schemaName, tableName, err := getSchemaTableNames(q)
	if err != nil {
		return err
	}

	r.ObjName = strings.Join([]string{schemaName, tableName}, ".")
	numRow, err := countRows(h.db, schemaName, tableName)
	if err != nil {
		return err
	}
	r.NumRow = numRow
	return nil
}

func (h *DBHandler) deleteRows(q *urlQuery, r *DBResult) error {
	schemaName, tableName, err := getSchemaTableNames(q)
	if err != nil {
		return err
	}

	r.ObjName = strings.Join([]string{schemaName, tableName}, ".")
	numRow, err := deleteRows(h.db, schemaName, tableName)
	if err != nil {
		return err
	}
	r.NumRow = numRow
	return nil
}

func (h *DBHandler) createTable(q *urlQuery, r *DBResult) error {
	schemaName, tableName, err := getSchemaTableNames(q)
	if err != nil {
		return err
	}

	r.ObjName = strings.Join([]string{schemaName, tableName}, ".")
	if err := createTable(h.db, schemaName, tableName); err != nil {
		return err
	}
	return nil
}

func (h *DBHandler) dropTable(q *urlQuery, r *DBResult) error {
	schemaName, tableName, err := getSchemaTableNames(q)
	if err != nil {
		return err
	}

	r.ObjName = strings.Join([]string{schemaName, tableName}, ".")
	if err := dropTable(h.db, schemaName, tableName); err != nil {
		return err
	}
	return nil
}

func (h *DBHandler) createSchema(q *urlQuery, r *DBResult) error {
	schemaName, err := q.get(urlQuerySchemaName)
	if err != nil {
		return err
	}
	r.ObjName = schemaName
	if err := createSchema(h.db, schemaName); err != nil {
		return err
	}
	return nil
}

func (h *DBHandler) dropSchema(q *urlQuery, r *DBResult) error {
	schemaName, err := q.get(urlQuerySchemaName)
	if err != nil {
		return err
	}
	r.ObjName = schemaName
	if err := dropSchema(h.db, schemaName, false); err != nil {
		return err
	}
	return nil
}
