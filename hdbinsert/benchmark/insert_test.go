// SPDX-FileCopyrightText: 2020 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/stfnmllr/go-hdb-test/hdbinsert/handler"
)

func BenchmarkInsert(b *testing.B) {

	checkErr := func(err error) {
		if err != nil {
			b.Fatal(err)
		}
	}

	// Create handlers.
	dbHandler, err := handler.NewDBHandler(b.Logf)
	checkErr(err)
	testHandler, err := handler.NewTestHandler(b.Logf)
	checkErr(err)

	// Register handlers.
	mux := http.NewServeMux()
	mux.Handle("/test/", testHandler)
	mux.Handle("/db/", dbHandler)

	// Start http test server.
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := ts.Client()

	execDB := func(cmd string) (*handler.DBResult, error) {
		r, err := client.Get(ts.URL + cmd)
		if err != nil {
			return nil, err
		}

		defer r.Body.Close()

		res := &handler.DBResult{}
		if err := json.NewDecoder(r.Body).Decode(res); err != nil {
			return nil, err
		}
		return res, nil
	}

	if execDB == nil {

	}

	execTest := func(test string) (*handler.TestResult, error) {
		r, err := client.Get(ts.URL + test)
		if err != nil {
			return nil, err
		}

		defer r.Body.Close()

		res := &handler.TestResult{}
		if err := json.NewDecoder(r.Body).Decode(res); err != nil {
			return nil, err
		}
		return res, nil
	}

	recreateTable := func() {
		// Drop and re-create table.
		// - delete rows lead to an out-of-memory error in HANA while trying do delete millions of rows.
		if _, err := execDB(handler.CmdDropTable); err != nil {
			b.Fatal(err)
		}
		if _, err := execDB(handler.CmdCreateTable); err != nil {
			b.Fatal(err)
		}
	}

	const maxDuration time.Duration = 1<<63 - 1

	f := func(test string, b *testing.B) {

		ds := make([]time.Duration, b.N)
		var avg, max time.Duration
		min := maxDuration

		for i := 0; i < b.N; i++ {
			r, err := execTest(test)
			if err != nil {
				b.Fatal(err)
			}
			if r.Error != "" {
				b.Fatal(r.Error)
			}

			avg += r.Duration
			if r.Duration < min {
				min = r.Duration
			}
			if r.Duration > max {
				max = r.Duration
			}
			ds[i] = r.Duration
		}

		// Median.
		var med time.Duration
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		l := len(ds)
		switch {
		case l == 0: // keep med == 0
		case l%2 != 0: // odd number
			med = ds[l/2] //  mid value
		default:
			med = (ds[l/2] + ds[l/2-1]) / 2 // even number - return avg of the two mid numbers
		}

		// Add metrics.
		b.ReportMetric((avg / time.Duration(b.N)).Seconds(), "avgsec/op")
		b.ReportMetric(min.Seconds(), "minsec/op")
		b.ReportMetric(max.Seconds(), "maxsec/op")
		b.ReportMetric(med.Seconds(), "medsec/op")
	}

	// Start benchmarks.
	tests := []string{handler.TestBulkSeq, handler.TestManySeq, handler.TestBulkPar, handler.TestManyPar}

	for _, test := range tests {
		recreateTable()
		// Use batchCount and batchCount flags.
		b.Run(test, func(b *testing.B) {
			f(test, b)
		})
	}
}
