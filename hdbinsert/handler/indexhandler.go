// SPDX-FileCopyrightText: 2020-2021 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"flag"
	"html/template"
	"net/http"
	"runtime"

	"github.com/stfnmllr/go-hdb-test/hdbinsert/env"
)

// IndexHandler implements the http.Handler interface for the html index page.
type IndexHandler struct {
	b *bytes.Buffer
}

// NewIndexHandler returns a new IndexHandler instance.
func NewIndexHandler(testHandler *TestHandler, dbHandler *DBHandler) (*IndexHandler, error) {
	return (&IndexHandler{b: new(bytes.Buffer)}).init(testHandler, dbHandler)
}

func (h *IndexHandler) init(testHandler *TestHandler, dbHandler *DBHandler) (*IndexHandler, error) {
	type page struct {
		GOMAXPROCS    int
		NumCPU        int
		DriverVersion string
		HDBVersion    string
		Flags         []*flag.Flag
		Prms          [][]env.Prm
		Tests         []string
		SchemaName    string
		TableName     string
		SchemaFuncs   []*dbFunc
		TableFuncs    []*dbFunc
	}

	indexPage := page{
		GOMAXPROCS:    runtime.GOMAXPROCS(0),
		NumCPU:        runtime.NumCPU(),
		DriverVersion: dbHandler.DriverVersion(),
		HDBVersion:    dbHandler.HDBVersion(),
		Flags:         env.Flags(),
		Prms:          env.Parameters().ToNumRecordList(),
		Tests:         testHandler.tests(),
		SchemaName:    env.SchemaName(),
		TableName:     env.TableName(),
		SchemaFuncs:   dbHandler.schemaFuncs(),
		TableFuncs:    dbHandler.tableFuncs(),
	}
	return h, indexTmpl.Execute(h.b, indexPage)
}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) { w.Write(h.b.Bytes()) }

var indexTmpl = template.Must(template.New("index").Parse(`
{{define "root"}}
<html>
	<head>
		<title>index</title>
	</head>
	<style>
		thead, tbody {
			border: 2px solid black;
			border-collapse: collapse;
		}
		
		table, th, td {
			border: 1px solid black;
			border-collapse: collapse;
		}
	</style>
	
	<body>
	
		<table border="1">
			<tr>	<th colspan="100%">Runtime information</td></tr>
			<tr>	<td>GOMAXPROCS</td><td>{{.GOMAXPROCS}}</td></tr>
			<tr>	<td>NumCPU</td><td>{{.NumCPU}}</td></tr>
			<tr>	<td>Driver Version</td><td>{{.DriverVersion}}</td></tr>
			<tr>	<td>HANA Version</td><td>{{.HDBVersion}}</td></tr>
		</table>

		<br/>
		
		<table border="1">
			<tr><th colspan="100%">Test parameter</td></tr>
			<tr>
				<th>Command line flag</td>
				<th>Value</td>
				<th>Usage</td>
			</tr>
			{{range .Flags}}
			<tr>
				<td>{{.Name}}</td>
				<td>{{.Value}}</td>
				<td>{{.Usage}}</td>
			</tr>
			{{end}}
		</table>

		<br/>
		
		<table border="1">
			<thead>
				<tr>
					<th rowspan="2">BatchCount x BatchSize</th>
					<th colspan="2">Sequential</th>
					<th colspan="2">Parallel</th>
				</tr>
				<tr>
					<th>bulk</th>
					<th>many</th>
					<th>bulk</th>
					<th>many</th>
				</tr>
			</thead>	
			{{$Tests := .Tests}}
			{{$Prms := .Prms}}
			{{range $PrmSet := $Prms}}
			</tbody>
			{{range $Prm := $PrmSet}}
			<tr>
				<td>{{$Prm.BatchCount}} x {{$Prm.BatchSize}}</td>
				{{range $Test := $Tests}}
				<td>{{with $x := printf "%s?batchcount=%d&batchsize=%d" $Test $Prm.BatchCount $Prm.BatchSize }}<a href={{$x}}>start</a>{{end}}</td>
				{{end}}
			</tr>
			{{end}}
			</tbody>
			{{end}}
		</table>

		<br/>
			
		<table border="1">
			{{$SchemaName := .SchemaName}}
			{{$TableName := .TableName}}
			<tr>
				<th colspan="100%">Database operations</td>
			</tr>
			<tr>
				<td>Table {{$TableName}}</td>
				{{range .TableFuncs}}
				{{$Op := .Op.String}}
				<td>{{with $x := printf "%s?schemaname=%s&tablename=%s" .Command $SchemaName $TableName }}<a href={{$x}}>{{$Op}}</a>{{end}}</td>
				{{end}}
			</tr>
			<tr>
				<td>Schema {{$SchemaName}}</td>
				{{range .SchemaFuncs}}
				{{$Op := .Op.String}}
				<td>{{with $x := printf "%s?schemaname=%s" .Command $SchemaName }}<a href={{$x}}>{{$Op}}</a>{{end}}</td>
				{{end}}
			</tr>
		</table>

	</body>
</html>
{{end}}

{{template "root" .}}
`))
