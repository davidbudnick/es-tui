package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Overridable in tests.
var (
	osExit     = os.Exit
	seedStdout = io.Writer(os.Stdout)
	seedStderr = io.Writer(os.Stderr)
)

func main() {
	if err := seedMain(os.Args[1:]); err != nil {
		fmt.Fprintf(seedStderr, "seed failed: %v\n", err)
		osExit(1)
		return
	}
	fmt.Fprintln(seedStdout, "Seed complete.")
}

func seedMain(args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	addr := fs.String("addr", "http://localhost:9200", "Cluster base URL")
	flush := fs.Bool("flush", false, "Delete demo indices before seeding")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return run(*addr, *flush)
}

func run(addr string, flush bool) error {
	client := &http.Client{Timeout: 30 * time.Second}
	indices := []string{"products", "orders", "logs-demo"}

	if flush {
		for _, idx := range indices {
			req, err := http.NewRequest(http.MethodDelete, addr+"/"+idx, nil)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}

	if err := put(client, addr+"/products", map[string]any{
		"settings": map[string]any{"number_of_shards": 1, "number_of_replicas": 0},
		"mappings": map[string]any{
			"properties": map[string]any{
				"name":  map[string]any{"type": "text"},
				"price": map[string]any{"type": "float"},
				"tags":  map[string]any{"type": "keyword"},
			},
		},
	}); err != nil {
		return fmt.Errorf("create products: %w", err)
	}

	products := []map[string]any{
		{"name": "Elastic Widget", "price": 19.99, "tags": []string{"search", "demo"}},
		{"name": "OpenSearch Gadget", "price": 24.50, "tags": []string{"search", "oss"}},
		{"name": "Kibana Mug", "price": 12.00, "tags": []string{"merch"}},
		{"name": "Dashboards Tee", "price": 28.00, "tags": []string{"merch", "oss"}},
	}
	for i, p := range products {
		if err := put(client, fmt.Sprintf("%s/products/_doc/%d", addr, i+1), p); err != nil {
			return err
		}
	}

	if err := put(client, addr+"/orders", map[string]any{
		"settings": map[string]any{"number_of_shards": 1, "number_of_replicas": 0},
	}); err != nil {
		return fmt.Errorf("create orders: %w", err)
	}
	orders := []map[string]any{
		{"order_id": "ORD-1001", "customer": "alice", "total": 44.99, "status": "shipped"},
		{"order_id": "ORD-1002", "customer": "bob", "total": 12.00, "status": "pending"},
		{"order_id": "ORD-1003", "customer": "carol", "total": 99.50, "status": "delivered"},
	}
	for i, o := range orders {
		if err := put(client, fmt.Sprintf("%s/orders/_doc/%d", addr, i+1), o); err != nil {
			return err
		}
	}

	if err := put(client, addr+"/logs-demo", map[string]any{
		"settings": map[string]any{"number_of_shards": 1, "number_of_replicas": 0},
	}); err != nil {
		return fmt.Errorf("create logs-demo: %w", err)
	}
	for i := 1; i <= 20; i++ {
		doc := map[string]any{
			"@timestamp": time.Now().Add(-time.Duration(i) * time.Minute).UTC().Format(time.RFC3339),
			"level":      []string{"info", "warn", "error"}[i%3],
			"message":    fmt.Sprintf("demo log line %d", i),
			"service":    "es-tui-seed",
		}
		if err := put(client, fmt.Sprintf("%s/logs-demo/_doc/%d", addr, i), doc); err != nil {
			return err
		}
	}

	req, err := http.NewRequest(http.MethodPost, addr+"/_refresh", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return nil
}

func put(client *http.Client, url string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return nil
}
