package main

import (
	"flag"
	"fmt"
	"infra/chromeperf/pinpoint/fakelegacy"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var (
	portFlag        = flag.Int("port", 1123, "Port to listen on; note that only loopback connections are supported")
	templateDirFlag = flag.String("template-dir", "", "Path to the response template directory")
)

func main() {
	flag.Parse()

	// TODO(chowski): once the infra repository moves to Go 1.16, we can just
	// embed the templates at compile time (https://golang.org/pkg/embed/).
	if *templateDirFlag == "" {
		log.Fatal("Must set --template-dir")
	}

	s, err := fakelegacy.NewServer(*templateDirFlag, nil)
	if err != nil {
		log.Fatalf("Failed to start fakelegacy server: %v", err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		log.Printf("Interrupt detected, exiting gracefully")
		os.Exit(0)
	}()

	hostport := fmt.Sprintf("localhost:%d", *portFlag)
	log.Printf("fakelegacy listening on %v", hostport)
	err = http.ListenAndServe(hostport, s.Handler())
	log.Fatal("fakelegacy failed: ", err)
}
