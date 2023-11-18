package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
)

type esutilOption struct {
	index string

	address string

	workers       int
	flushBytes    int
	flushInterval time.Duration
}

type Document map[string]interface{}

func (d *Document) DocID() string {
	fields := []string{
		"_id",
		"id",
		"ID",
		"Id",
	}
	for _, field := range fields {
		if _, ok := (*d)[field]; !ok {
			continue
		}
		switch (*d)[field].(type) {
		case int:
			return fmt.Sprintf("%d", (*d)[field].(int))
		case int64:
			return fmt.Sprintf("%d", (*d)[field].(int64))
		case float64:
			return fmt.Sprintf("%.0f", (*d)[field].(float64))
		case string:
			return (*d)[field].(string)
		default:
			panic(fmt.Sprintf("unsupported doc id type: %T", (*d)[field]))
		}
	}
	panic("doc_id not found")
}

func (d *Document) Body() io.ReadSeeker {
	body, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(body))
	return bytes.NewReader(body)
}

func main() {
	opt := esutilOption{
		index:         "",
		address:       "http://localhost:9200",
		workers:       runtime.NumCPU(),
		flushBytes:    5e+6,
		flushInterval: time.Second * 30,
	}

	flag.StringVar(&opt.index, "index", opt.index, "elasticsearch index")
	flag.StringVar(&opt.address, "address", opt.address, "elasticsearch address")
	flag.IntVar(&opt.workers, "workers", opt.workers, "num workers")
	flag.IntVar(&opt.flushBytes, "flush-bytes", opt.flushBytes, "flush bytes threshold")
	flag.DurationVar(&opt.flushInterval, "flush-interval", opt.flushInterval, "flush interval")
	flag.Parse()

	if opt.index == "" {
		log.Fatal("index is required")
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{opt.address},
		Username:  "elastic",
		Password:  "password",
	})
	if err != nil {
		log.Fatal(err)
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         opt.index,
		Client:        es,
		NumWorkers:    opt.workers,
		FlushBytes:    opt.flushBytes,
		FlushInterval: opt.flushInterval,
	})
	if err != nil {
		log.Fatal(err)
	}

	inputs, err := readDocuments(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	count := int64(0)
	for _, input := range inputs {
		record := NewUpdateRecord(&input)
		if err := bi.Add(context.Background(), esutil.BulkIndexerItem{
			Action:     "update",
			DocumentID: record.DocID(),
			Body:       record.Body(),
			OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
				atomic.AddInt64(&count, 1)
			},
			OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
				if err != nil {
					log.Printf("ERROR: %s", err)
				} else {
					log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
				}
			},
		}); err != nil {
			log.Fatal(err)
		}
	}
	if err := bi.Close(context.Background()); err != nil {
		log.Fatal(err)
	}

	biStats := bi.Stats()

	// Report the results: number of indexed docs, number of errors, duration, indexing rate
	//
	dur := time.Since(start)
	log.Println(strings.Repeat("▔", 65))
	if biStats.NumFailed > 0 {
		log.Fatalf(
			"Indexed [%s] documents with [%s] errors in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			humanize.Comma(int64(biStats.NumFailed)),
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
	} else {
		log.Printf(
			"Sucessfuly indexed [%s] documents in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
	}
}

func readDocuments(input io.Reader) ([]Document, error) {
	result := make([]Document, 0)
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var r map[string]interface{}
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}
