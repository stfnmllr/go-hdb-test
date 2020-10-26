// SPDX-FileCopyrightText: 2020 Stefan Miller
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/stfnmllr/go-hdb-test/hdbinsert/env"
	"github.com/stfnmllr/go-hdb-test/hdbinsert/handler"

	// Add profiling.
	_ "net/http/pprof"
)

func main() {
	if !flag.Parsed() {
		flag.Parse()
	}

	checkErr := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	// Print runtime info.
	log.Printf("Runtime Info - GOMAXPROCS: %d NumCPU: %d", runtime.GOMAXPROCS(0), runtime.NumCPU())

	s := make([]string, 0)
	env.Visit(func(f *flag.Flag) {
		s = append(s, fmt.Sprintf("%s:%s", f.Name, f.Value))
	})
	log.Printf("Command line flags: %s", strings.Join(s, " "))

	// Create handlers.
	dbHandler, err := handler.NewDBHandler(log.Printf)
	checkErr(err)
	testHandler, err := handler.NewTestHandler(log.Printf)
	checkErr(err)
	indexHandler, err := handler.NewIndexHandler(testHandler, dbHandler)
	checkErr(err)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	mux := http.NewServeMux()

	mux.Handle("/test/", testHandler)
	mux.Handle("/db/", dbHandler)
	mux.Handle("/", indexHandler)
	mux.HandleFunc("/favicon.ico", func(http.ResponseWriter, *http.Request) {}) // Avoid "/" handler call for browser favicon request.

	svr := http.Server{Addr: net.JoinHostPort(env.Host(), env.Port()), Handler: mux}
	log.Println("listening...")

	go func() {
		if err := svr.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-sigint
	// shutdown server
	log.Println("shutting down...")
	if err := svr.Shutdown(context.Background()); err != nil {
		log.Fatalf("HTTP server Shutdown: %v", err)
	}
}
