package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var verbose = flag.Bool("v", false, "log more stuff")
var retries = flag.Int("retries", 3, "how many times to retry each post")
var backoff = flag.Duration("retrytime", 30*time.Second,
	"How long to wait between retries")

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %v [flags] url...\n",
			os.Args[0])
		flag.PrintDefaults()
		os.Exit(64)
	}
}

type result struct {
	u   string
	err error
}

func process(u string, in []byte, ch chan result) {
	var latestError error
	for i := 0; i < *retries; i++ {
		latestError = nil

		if *verbose {
			log.Printf("Trying %v", u)
		}

		req, err := http.NewRequest("POST", u, bytes.NewReader(in))
		if err != nil {
			log.Fatalf("Error creating request for %v: %v", u, err)
		}

		res, err := http.DefaultClient.Do(req)
		if err == nil {
			res.Body.Close()
			if res.StatusCode < 200 || res.StatusCode >= 300 {
				err = fmt.Errorf("http error: %v", res.Status)
			}
		}
		if err != nil {
			latestError = err
			log.Printf("Error on %v: %v", u, err)
			time.Sleep(*backoff)
			continue
		}
		break
	}

	ch <- result{u, latestError}
}

func main() {
	flag.Parse()

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Error reading stdin: %v", err)
	}

	if flag.NArg() == 0 {
		flag.Usage()
	}

	ch := make(chan result)

	for _, u := range flag.Args() {
		go process(u, input, ch)
	}

	errors := 0
	for _ = range flag.Args() {
		r := <-ch

		if r.err != nil {
			log.Printf("Error on %v: %v", r.u, r.err)
			errors++
		}
	}

	if errors > 0 {
		log.Fatalf("There were %v errors", errors)
	}
}
