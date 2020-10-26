// SPDX-FileCopyrightText: 2020 Stefan Miller
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
	type prm struct {
		Count, Size int
	}

	type page struct {
		GOMAXPROCS  int
		NumCPU      int
		Flags       []*flag.Flag
		Prms        [][]prm
		Tests       []string
		SchemaName  string
		TableName   string
		SchemaFuncs []*dbFunc
		TableFuncs  []*dbFunc
	}

	indexPage := page{
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		NumCPU:     runtime.NumCPU(),
		Flags:      env.Flags(),
		Prms: [][]prm{
			{{1, 100000}, {10, 10000}, {100, 1000}},
			{{1, 1000000}, {10, 100000}, {100, 10000}, {1000, 1000}},
		},
		Tests:       testHandler.tests(),
		SchemaName:  env.SchemaName(),
		TableName:   env.TableName(),
		SchemaFuncs: dbHandler.schemaFuncs(),
		TableFuncs:  dbHandler.tableFuncs(),
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
				<td>{{$Prm.Count}} x {{$Prm.Size}}</td>
				{{range $Test := $Tests}}
				<td>{{with $x := printf "%s?batchcount=%d&batchsize=%d" $Test $Prm.Count $Prm.Size }}<a href={{$x}}>start</a>{{end}}</td>
				{{end}}
			</tr>
			{{end}}
			</tbody>
			{{end}}
		</table>

		<br/>
			
		<table border="1">
			<tr>
				<th colspan="100%">Database operations</td>
			</tr>
			<tr>
				<td>Table {{.TableName}}</td>
				{{range .TableFuncs}}
				<td><a href={{.Command}}>{{.Op.String}}</a></td>
				{{end}}
			</tr>
			<tr>
				<td>Schema {{.SchemaName}}</td>
				{{range .SchemaFuncs}}
				<td><a href={{.Command}}>{{.Op.String}}</a></td>
				{{end}}
			</tr>
		</table>

	</body>
</html>
{{end}}

{{template "root" .}}
`))
