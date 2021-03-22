// SPDX-FileCopyrightText: 2020-2021 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"database/sql"
	"fmt"

	"github.com/SAP/go-hdb/driver"
)

// createSchema creates a schema on the database.
func createSchema(db *sql.DB, name string) error {
	_, err := db.Exec(fmt.Sprintf("create schema %s", driver.Identifier(name)))
	return err
}

// dropSchema drops a schema from the database even if the schema is not empty.
func dropSchema(db *sql.DB, name string, cascade bool) error {
	var stmt string
	if cascade {
		stmt = fmt.Sprintf("drop schema %s cascade", driver.Identifier(name))
	} else {
		stmt = fmt.Sprintf("drop schema %s", driver.Identifier(name))
	}
	_, err := db.Exec(stmt)
	return err
}

// createTable creates a table from the databasesde.
func createTable(db *sql.DB, schemaName, tableName string) error {
	_, err := db.Exec(fmt.Sprintf("create column table %s.%s (%s)", driver.Identifier(schemaName), driver.Identifier(tableName), columns))
	return err
}

// dropTable drops a table from the databases.
func dropTable(db *sql.DB, schemaName, tableName string) error {
	_, err := db.Exec(fmt.Sprintf("drop table %s.%s", driver.Identifier(schemaName), driver.Identifier(tableName)))
	return err
}

// existTable returns true if the table exists in schema.
func existTable(db *sql.DB, schemaName, tableName string) (bool, error) {
	numTables := 0
	if err := db.QueryRow(fmt.Sprintf("select count(*) from sys.tables where schema_name = '%s' and table_name = '%s'", schemaName, tableName)).Scan(&numTables); err != nil {
		return false, err
	}
	return numTables != 0, nil
}

// ensureTable creates a table if it does not exist. If drop is set, an existing table would be dropped before recreated.
func ensureTable(db *sql.DB, schemaName, tableName string, drop bool) error {
	exist, err := existTable(db, schemaName, tableName)
	if err != nil {
		return err
	}

	switch {
	case exist && drop:
		if err := dropTable(db, schemaName, tableName); err != nil {
			return err
		}
		if err := createTable(db, schemaName, tableName); err != nil {
			return err
		}
	case !exist:
		if err := createTable(db, schemaName, tableName); err != nil {
			return err
		}
	}
	return nil
}

// deleteRows deletes all records in the database table.
func deleteRows(db *sql.DB, schemaName, tableName string) (int64, error) {
	result, err := db.Exec(fmt.Sprintf("delete from %s.%s", driver.Identifier(schemaName), driver.Identifier(tableName)))
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
func countRows(db *sql.DB, schemaName, tableName string) (int64, error) {
	var numRow int64

	err := db.QueryRow(fmt.Sprintf("select count(*) from %s.%s", driver.Identifier(schemaName), driver.Identifier(tableName))).Scan(&numRow)
	if err != nil {
		return 0, err
	}
	return numRow, nil
}
