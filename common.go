package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	requests    int
	concurrency int
	timelimit   int

	method              string
	bodyFile            string
	contentType         string
	headers             []string
	cookies             []string
	gzip                bool
	keepAlive           bool
	basicAuthentication string
	userAgent           string

	url  string
	host string
	port int

	serverName  string
	contentSize int
}

func loadConfig() (config *Config, err error) {
	flag.IntVar(&Verbosity, "v", 0, "How much troubleshooting info to print")
	flag.IntVar(&GoMaxProcs, "G", runtime.NumCPU(), "Number of Goroutine procs")
	flag.BoolVar(&ContinueOnError, "r", false, "Don't exit on socket receive errors")

	request := flag.Int("n", 1, "Number of requests to perform")
	concurrency := flag.Int("c", 1, "Number of multiple requests to make")
	timelimit := flag.Int("t", 0, "Seconds to max. wait for responses")

	postFile := flag.String("p", "", "File containing data to POST. Remember also to set -T")
	putFile := flag.String("u", "", "File containing data to PUT. Remember also to set -T")
	contentType := flag.String("T", "text/plain", "Content-type header for POSTing, eg. 'application/x-www-form-urlencoded' Default is 'text/plain'")

	var headers, cookies stringSet
	flag.Var(&headers, "H", "Add Arbitrary header line, eg. 'Accept-Encoding: gzip' Inserted after all normal header lines. (repeatable)")
	flag.Var(&cookies, "C", "Add cookie, eg. 'Apache=1234. (repeatable)")

	basicAuthentication := flag.String("A", "", "Add Basic WWW Authentication, the attributes are a colon separated username and password.")

	keepAlive := flag.Bool("k", false, "Use HTTP KeepAlive feature")
	gzip := flag.Bool("z", false, "Use HTTP Gzip feature")
	help := flag.Bool("h", false, "Display usage information (this message)")

	flag.Parse()

	flag.Usage = func() {
		fmt.Print("Usage: gb [options] [http[s]://]hostname[:port]/path\nOptions are:\n")
		flag.PrintDefaults()
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	config = &Config{}
	config.requests = *request
	config.concurrency = *concurrency

	if *postFile == "" || *putFile == "" {
		config.method = "GET"
	} else if *postFile != "" {
		config.method = "POST"
		config.bodyFile = *postFile

	} else if *putFile != "PUT" {
		config.method = "PUT"
		config.bodyFile = *putFile
	}

	if *timelimit > 0 {
		config.timelimit = *timelimit
		if config.requests == 1 {
			config.requests = MAX_REQUESTS
		}
	}

	config.contentType = *contentType
	config.keepAlive = *keepAlive
	config.gzip = *gzip
	config.basicAuthentication = *basicAuthentication

	config.url = os.Args[len(os.Args)-1]
	URL, err := url.Parse(config.url)
	if err != nil {
		return
	}
	config.host, config.port = extractHostAndPort(URL)

	if config.requests < 1 || config.concurrency < 1 || config.timelimit < 0 || GoMaxProcs < 1 || Verbosity < 0 {
		err = errors.New("wrong number of arguments")
	}

	if config.concurrency > config.requests {
		err = errors.New("Cannot use concurrency level greater than total number of requests")

	}

	return
}

type stringSet []string

func (f *stringSet) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *stringSet) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func extractHostAndPort(url *url.URL) (host string, port int) {

	hostname := url.Host
	pos := strings.LastIndex(hostname, ":")
	if pos > 0 {
		host = hostname[0:pos]
		portInt64, err := strconv.Atoi(hostname[pos+1:])
		if err != nil {
			log.Fatal(err)
		}
		port = int(portInt64)
	} else {
		host = hostname
		if url.Scheme == "http" {
			port = 80
		} else if url.Scheme == "https" {
			port = 443
		}
	}
	return
}

type StopWatch struct {
	start   time.Time
	end     time.Time
	Elapsed time.Duration
}

func (s *StopWatch) Start() {
	s.start = time.Now()
}

func (s *StopWatch) Stop() {
	s.end = time.Now()
	s.Elapsed = s.end.Sub(s.start)
}

func TraceException(msg interface{}) {

	switch {
	case Verbosity > 1:
		var buffer bytes.Buffer
		buffer.WriteString(fmt.Sprintf("recover: %v\n", msg))
		for skip := 1; ; skip++ {
			pc, file, line, ok := runtime.Caller(skip)
			if !ok {
				break
			}
			f := runtime.FuncForPC(pc)
			buffer.WriteString(fmt.Sprintf("\t%s:%d %s()\n", file, line, f.Name()))
		}
		buffer.WriteString("\n")
		fmt.Fprint(os.Stderr, buffer.String())
	case Verbosity > 0:
		fmt.Fprintf(os.Stderr, "recover: %v\n", msg)
	}
}
