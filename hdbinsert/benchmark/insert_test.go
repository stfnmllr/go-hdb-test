// SPDX-FileCopyrightText: 2020-2021 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package benchmark

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/stfnmllr/go-hdb-test/hdbinsert/env"
	"github.com/stfnmllr/go-hdb-test/hdbinsert/handler"
)

func BenchmarkInsert(b *testing.B) {

	checkErr := func(err error) {
		if err != nil {
			b.Fatal(err)
		}
	}

	// Create handler.
	testHandler, err := handler.NewTestHandler(b.Logf)
	checkErr(err)
	dbHandler, err := handler.NewDBHandler(b.Logf)
	checkErr(err)

	// Register handlers.
	mux := http.NewServeMux()
	mux.Handle("/test/", testHandler)

	// Start http test server.
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := ts.Client()

	execTest := func(test string, batchCount, batchSize int) (*handler.TestResult, error) {
		r, err := client.Get(fmt.Sprintf("%s%s?batchcount=%d&batchsize=%d", ts.URL, test, batchCount, batchSize))
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

	const maxDuration time.Duration = 1<<63 - 1

	f := func(test string, batchCount, batchSize int, b *testing.B) {
		ds := make([]time.Duration, b.N)
		var avg, max time.Duration
		min := maxDuration

		for i := 0; i < b.N; i++ {
			r, err := execTest(test, batchCount, batchSize)
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

	// Additional info.
	log.SetOutput(os.Stdout)

	format := `
GOMAXPROCS: %d
NumCPU: %d
Driver Version: %s
HANA Version: %s
`
	log.Printf(format, runtime.GOMAXPROCS(0), runtime.NumCPU(), dbHandler.DriverVersion(), dbHandler.HDBVersion())

	// Start benchmarks.
	names := []string{"bulkSeq", "manySeq", "bulkPar", "manyPar"}
	tests := []string{handler.TestBulkSeq, handler.TestManySeq, handler.TestBulkPar, handler.TestManyPar}

	for _, prm := range env.Parameters().Prms {
		b.Run(fmt.Sprintf("%dx%d", prm.BatchCount, prm.BatchSize), func(b *testing.B) {
			for i, test := range tests {
				// Use batchCount and batchCount flags.
				b.Run(names[i], func(b *testing.B) {
					f(test, prm.BatchCount, prm.BatchSize, b)
				})
			}
		})
	}
}
