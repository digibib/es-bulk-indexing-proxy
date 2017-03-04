package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const MB = 1048576

type indexReq struct {
	typ  string // PUT | DELETE
	id   string
	body []byte
}

type proxy struct {
	es        string
	esProxy   *httputil.ReverseProxy
	indexReqs chan indexReq
	bulkSize  float64 // * MB
	maxWait   time.Duration
}

func (p *proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if i := strings.Index(req.URL.Path, "http%3A"); i != -1 && (req.Method == "PUT" || req.Method == "DELETE") {
		idx := indexReq{typ: req.Method, id: req.URL.Path[i:]}
		var err error
		idx.body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		go func() {
			p.indexReqs <- idx
		}()

		rw.WriteHeader(http.StatusAccepted)
		return
	}

	// Dispatch non-indexing requests to Elasticsearch
	p.esProxy.ServeHTTP(rw, req)
}

func typeFromId(id string) string {
	id = strings.TrimPrefix(id, "http%3A%2F%2Fdata.deichman.no%2F")
	i := strings.Index(id, "%")
	return id[:i]
}

func (p *proxy) handleBatch() {
	var b bytes.Buffer
	n := 0 // size in bytes
	d := 0 // number of index/delete requests
	for {

		select {
		case i := <-p.indexReqs:
			d++
			// Write action header
			if i.typ == "PUT" {
				b.WriteString(`{"index":{"_index":"search","_type":"`)
			} else {
				b.WriteString(`{"delete":{"_index":"search","_type":"`)
			}
			b.WriteString(typeFromId(i.id))
			b.WriteString(`","_id":"`)
			b.WriteString(i.id)
			b.WriteString("\"}}\n")

			// Write body
			nBody, _ := b.Write(i.body)
			n += nBody
			b.Write([]byte("\n"))
			n += 100 // TODO size of header
			if float64(n) >= p.bulkSize*MB {
				break
			}
			continue
		case <-time.After(p.maxWait):
			if n == 0 {
				continue
			}
			break
		}

		// Send batch to Elasticsearch
		log.Printf("Sending batch of %d requests to Elasticsearch", d)
		resp, err := http.Post(p.es+"/_bulk", "application/json", &b)
		if err != nil {
			log.Printf("Sending batch to Elasticsearch failed: %v", err)
		} else {
			log.Println("OK")
			resp.Body.Close()
		}

		b.Reset()
		d = 0
		n = 0
	}
}
func main() {
	var (
		l       = flag.String("l", ":9200", "Listening address")
		es      = flag.String("es", "http://elasticsearch:9200", "Elasticsearch proxy target")
		maxWait = flag.Duration("max-wait", 10*time.Second, "Max duration of inactivity before dispatch")
		mbSize  = flag.Float64("size", 1, "Batch size in number of MB")
	)
	flag.Parse()

	esURL, err := url.Parse(*es)
	if err != nil {
		log.Fatal(err)
	}

	p := &proxy{
		es:        strings.TrimSuffix(*es, "/"),
		esProxy:   httputil.NewSingleHostReverseProxy(esURL),
		bulkSize:  *mbSize,
		maxWait:   *maxWait,
		indexReqs: make(chan indexReq),
	}

	go p.handleBatch()

	if err := http.ListenAndServe(*l, p); err != nil {
		log.Fatal(err)
	}
}
