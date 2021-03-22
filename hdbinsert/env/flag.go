// SPDX-FileCopyrightText: 2020-2021 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/SAP/go-hdb/driver"
)

// Flag name constants.
const (
	FnDSN        = "dsn"
	FnHost       = "host"
	FnPort       = "port"
	FnSchemaName = "schemaName"
	FnTableName  = "tableName"
	FnBufferSize = "bufferSize"
	FnParameters = "parameters"
	FnDrop       = "drop"
	FnSeparate   = "separate"
	FnWait       = "wait"
)

var flagNames = []string{FnDSN, FnHost, FnPort, FnSchemaName, FnTableName, FnBufferSize, FnParameters, FnDrop, FnSeparate, FnWait}

// Environment constants.
const (
	envDSN        = "GOHDBDSN"
	envHost       = "HOST"
	envPort       = "PORT"
	envSchemaName = "SCHEMANAME"
	envTableName  = "TABLENAME"
	envBufferSize = "BUFFERSIZE"
	envParameters = "PARAMETERS"
	envDrop       = "DROP"
	envSeparate   = "SEPARATE"
	envWait       = "WAIT"
)

var (
	dsn, host, port       string
	schemaName, tableName string
	bufferSize            int
	parameters            = &PrmValue{Prms: []Prm{{1, 100000}, {10, 10000}, {100, 1000}, {1, 1000000}, {10, 100000}, {100, 10000}, {1000, 1000}}}
	drop, separate        bool
	wait                  int
)

var initRan bool

func init() {
	if initRan {
		return
	}
	initRan = true

	flag.StringVar(&dsn, FnDSN, getStringEnv(envDSN, "hdb://MyUser:MyPassword@localhost:39013"), fmt.Sprintf("DNS (environment variable: %s)", envDSN))
	flag.StringVar(&host, FnHost, getStringEnv(envHost, "localhost"), fmt.Sprintf("HTTP host (environment variable: %s)", envHost))
	flag.StringVar(&port, FnPort, getStringEnv(envPort, "8080"), fmt.Sprintf("HTTP port (environment variable: %s)", envPort))
	flag.StringVar(&schemaName, FnSchemaName, getStringEnv(envSchemaName, "TG20POC"), fmt.Sprintf("Schema name (environment variable: %s)", envSchemaName))
	flag.StringVar(&tableName, FnTableName, getStringEnv(envTableName, "GOMESSAGE"), fmt.Sprintf("Table name (environment variable: %s)", envTableName))
	flag.IntVar(&bufferSize, FnBufferSize, getIntEnv(envBufferSize, driver.DefaultBufferSize), fmt.Sprintf("Buffer size in bytes (environment variable: %s)", envBufferSize))
	flag.Var(parameters, FnParameters, fmt.Sprintf("Parameters (environment variable: %s)", envParameters))
	flag.BoolVar(&drop, FnDrop, getBoolEnv(envDrop, true), fmt.Sprintf("Drop table before test (environment variable: %s)", envDrop))
	flag.BoolVar(&separate, FnSeparate, getBoolEnv(envSeparate, false), fmt.Sprintf("Separate tables for parallel tests (environment variable: %s)", envSeparate))
	flag.IntVar(&wait, FnWait, getIntEnv(envWait, 0), fmt.Sprintf("Wait time before starting test in seconds (environment variable: %s)", envWait))
}

// DSN returns the dsn command-line flag.
func DSN() string { return dsn }

// Host returns the host command-line flag.
func Host() string { return host }

// Port returns the port command-line flag.
func Port() string { return port }

// SchemaName returns the schemaName command-line flag.
func SchemaName() string { return schemaName }

// TableName returns the tableName command-line flag.
func TableName() string { return tableName }

// BufferSize returns the bufferSize command-line flag.
func BufferSize() int { return bufferSize }

// Parameters return the parameters command-line flag.
func Parameters() *PrmValue { return parameters }

// Drop returns the drop command-line flag.
func Drop() bool { return drop }

// Separate returns the separate command-line flag.
func Separate() bool { return separate }

// Wait returns the wait command-line flag.
func Wait() int { return wait }

// Flags returns a slice containing all command-line flags defined in this package.
func Flags() []*flag.Flag {
	flags := make([]*flag.Flag, 0)
	for _, name := range flagNames {
		if fl := flag.Lookup(name); fl != nil {
			flags = append(flags, fl)
		}
	}
	return flags
}

// Visit visits the command-line flags defined in this package.
func Visit(f func(f *flag.Flag)) {
	for _, fl := range Flags() {
		f(fl)
	}
}

// getStringEnv retrieves the string value of the environment variable named by the key.
// If the variable is present in the environment the value is returned.
// Otherwise the default value  defValue is retuned.
func getStringEnv(key, defValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defValue
	}
	return value
}

// getIntEnv retrieves the int value of the environment variable named by the key.
// If the variable is present in the environment the value is returned.
// Otherwise the default value defValue is retuned.
func getIntEnv(key string, defValue int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defValue
	}
	return i
}

// getBoolEnv retrieves the bool value of the environment variable named by the key.
// If the variable is present in the environment the value is returned.
// Otherwise the default value defValue is retuned.
func getBoolEnv(key string, defValue bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defValue
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return defValue
	}
	return b
}
