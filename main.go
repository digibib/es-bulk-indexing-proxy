package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type proxy struct {
	esProxy *httputil.ReverseProxy
}

func (p *proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// If index request queue up
	// TODO
	// Else dispatch to ES directly
	p.esProxy.ServeHTTP(rw, req)
}

func main() {
	var (
		l  = flag.String("l", ":9200", "Listening address")
		es = flag.String("es", "http://elasticsearch:9200", "Elasticsearch proxy target")
		//maxWait = flag.Duration("max-wait", 10*time.Second, "Max duration of inactivity before dispatch")
		//mbSize  = flag.Float64("size", 1, "Batch size in number of MB")
	)
	flag.Parse()

	esURL, err := url.Parse(*es)
	if err != nil {
		log.Fatal(err)
	}

	p := &proxy{
		esProxy: httputil.NewSingleHostReverseProxy(esURL),
	}

	if err := http.ListenAndServe(*l, p); err != nil {
		log.Fatal(err)
	}
}
