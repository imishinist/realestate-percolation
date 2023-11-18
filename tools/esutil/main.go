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

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type RealEstate struct {
	ID       int       `json:"id,omitempty"`
	Name     string    `json:"name"`
	Location *Location `json:"location,omitempty"`
}

func (r *RealEstate) DocID() string {
	return fmt.Sprintf("%d", r.ID)
}

func (r *RealEstate) Body() io.ReadSeeker {
	body, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
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

	inputs, err := readRealEstates(os.Stdin)
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

func readRealEstates(input io.Reader) ([]RealEstate, error) {
	result := make([]RealEstate, 0)
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var r RealEstate
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}
