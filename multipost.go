package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var verbose = flag.Bool("v", false, "log more stuff")
var retries = flag.Int("retries", 3, "how many times to retry each post")
var backoff = flag.Duration("retrytime", 30*time.Second,
	"How long to wait between retries")
var fromFile = flag.String("input", "-", "File from which to read body")
var paramName = flag.String("param", "payload", "Parameter name")
var timeLimit = flag.Duration("timeLimit", 15*time.Minute,
	"The maximum amount of time this process may run")

var headerProto = http.Header{}

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

		for k, v := range headerProto {
			req.Header[k] = v
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

func getInput() ([]byte, error) {
	var input []byte

	var f io.Reader
	if *fromFile == "-" {
		f = os.Stdin
	} else {
		ff, err := os.Open(*fromFile)
		if err != nil {
			return nil, err
		}
		defer ff.Close()
		f = ff
	}

	var err error
	input, err = ioutil.ReadAll(f)
	if err != nil {
		return input, err
	}

	if *paramName != "" {
		headerProto.Set("Content-Type", "application/x-www-form-urlencoded")
		escaped := url.QueryEscape(string(input))
		input = []byte(*paramName + "=" + escaped)
	}

	return input, nil
}

func main() {
	flag.Parse()

	time.AfterFunc(*timeLimit, func() {
		log.Fatalf("Reached absolute time limit")
	})

	input, err := getInput()
	if err != nil {
		log.Fatalf("Error acquiring input: %v", err)
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
