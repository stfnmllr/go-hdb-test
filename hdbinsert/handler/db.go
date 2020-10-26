// SPDX-FileCopyrightText: 2020 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

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
	f       func() (int64, error)
}

// DBHandler implements the http.Handler interface for database operations.
type DBHandler struct {
	log                   logFunc
	db                    *sql.DB
	schemaName, tableName driver.Identifier
	columns               string
	dbFuncs               map[string]*dbFunc
}

// NewDBHandler returns a new DBHandler instance.
func NewDBHandler(log logFunc) (*DBHandler, error) {
	connector, err := driver.NewConnector(map[string]interface{}{"dsn": env.DSN(), "defaultSchema": env.SchemaName()})
	if err != nil {
		return nil, err
	}
	h := &DBHandler{log: log, db: sql.OpenDB(connector), schemaName: driver.Identifier(env.SchemaName()), tableName: driver.Identifier(env.TableName()), columns: columns}
	h.dbFuncs = map[string]*dbFunc{
		CmdCountRows:    {Command: CmdCountRows, Obj: objTable, Op: opCountRows, f: h.countRows},
		CmdDeleteRows:   {Command: CmdDeleteRows, Obj: objTable, Op: opDeleteRows, f: h.deleteRows},
		CmdCreateTable:  {Command: CmdCreateTable, Obj: objTable, Op: opCreate, f: func() (int64, error) { err := h.createTable(); return -1, err }},
		CmdDropTable:    {Command: CmdDropTable, Obj: objTable, Op: opDrop, f: func() (int64, error) { err := h.dropTable(); return -1, err }},
		CmdCreateSchema: {Command: CmdCreateSchema, Obj: objSchema, Op: opCreate, f: func() (int64, error) { err := h.createSchema(); return -1, err }},
		CmdDropSchema:   {Command: CmdDropSchema, Obj: objSchema, Op: opDrop, f: func() (int64, error) { err := h.dropSchema(false); return -1, err }},
	}
	return h, nil
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
	var numRow int64

	dbFunc, ok := h.dbFuncs[command]
	if ok {
		numRow, err = dbFunc.f()
	} else {
		err = fmt.Errorf("Invalid command %s", command)
	}

	result.DbObj = dbFunc.Obj
	result.DbOp = dbFunc.Op
	switch dbFunc.Obj {
	case objTable:
		result.ObjName = string(h.tableName)
	case objSchema:
		result.ObjName = string(h.schemaName)
	}
	result.NumRow = numRow
	if err != nil {
		result.Error = err.Error()
	}
}

// createSchema creates a schema on the database.
func (h *DBHandler) createSchema() error {
	_, err := h.db.Exec(fmt.Sprintf("create schema %s", h.schemaName))
	return err
}

// dropSchema drops a schema from the database even if the schema is not empty.
func (h *DBHandler) dropSchema(cascade bool) error {
	var stmt string
	if cascade {
		stmt = fmt.Sprintf("drop schema %s cascade", h.schemaName)
	} else {
		stmt = fmt.Sprintf("drop schema %s", h.schemaName)
	}
	_, err := h.db.Exec(stmt)
	return err
}

// createTable creates a table from the databasesde.
func (h *DBHandler) createTable() error {
	_, err := h.db.Exec(fmt.Sprintf("create column table %s (%s)", h.tableName, columns))
	return err
}

// dropTable drops a table from the databases.
func (h *DBHandler) dropTable() error {
	_, err := h.db.Exec(fmt.Sprintf("drop table %s", h.tableName))
	return err
}

// deleteTable deletes all records in the database table.
func (h *DBHandler) deleteRows() (int64, error) {
	result, err := h.db.Exec(fmt.Sprintf("delete from %s", h.tableName))
	if err != nil {
		return 0, err
	}
	numRow, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return numRow, nil
}

// countRows returns the number of rows in the database table.
func (h *DBHandler) countRows() (int64, error) {
	var numRow int64

	err := h.db.QueryRow(fmt.Sprintf("select count(*) from %s", h.tableName)).Scan(&numRow)
	if err != nil {
		return 0, err
	}
	return numRow, nil
}
