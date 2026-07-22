// Command seed populates Elasticsearch/OpenSearch instances from the example
// docker-compose files with realistic demo indices for exercising es-tui.
//
//	go run ./examples/seed -flush                         # ES :9200
//	go run ./examples/seed -addr http://localhost:9201 -flush  # OpenSearch
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
	fmt.Fprintln(seedStdout, "Done — seeding complete")
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
	client := &http.Client{Timeout: 60 * time.Second}
	addr = strings.TrimRight(addr, "/")

	indices := demoIndexNames()
	if flush {
		fmt.Fprintf(seedStdout, "Flushing demo indices on %s\n", addr)
		for _, idx := range indices {
			_ = del(client, addr+"/"+idx)
		}
		_ = del(client, addr+"/_alias/shop")
	}

	fmt.Fprintf(seedStdout, "Seeding %s\n", addr)

	if err := seedProducts(client, addr); err != nil {
		return err
	}
	if err := seedCustomers(client, addr); err != nil {
		return err
	}
	if err := seedOrders(client, addr); err != nil {
		return err
	}
	if err := seedLogs(client, addr); err != nil {
		return err
	}
	if err := seedMetrics(client, addr); err != nil {
		return err
	}
	if err := seedEvents(client, addr); err != nil {
		return err
	}

	if err := post(client, addr+"/_aliases", map[string]any{
		"actions": []map[string]any{
			{"add": map[string]any{"index": "products", "alias": "shop", "is_write_index": true}},
			{"add": map[string]any{"index": "products", "alias": "catalog"}},
		},
	}); err != nil {
		fmt.Fprintf(seedStdout, "  aliases:      skipped (%v)\n", err)
	} else {
		fmt.Fprintln(seedStdout, "  aliases:      shop, catalog → products")
	}

	if err := post(client, addr+"/_refresh", nil); err != nil {
		return fmt.Errorf("refresh: %w", err)
	}
	return nil
}

func demoIndexNames() []string {
	// includes legacy logs-demo so -flush cleans old seeds too
	return []string{"products", "customers", "orders", "logs-app", "logs-demo", "metrics-host", "events"}
}

func seedProducts(client *http.Client, addr string) error {
	if err := put(client, addr+"/products", map[string]any{
		"settings": map[string]any{
			"number_of_shards":   1,
			"number_of_replicas": 0,
			"refresh_interval":   "1s",
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"sku":         map[string]any{"type": "keyword"},
				"name":        map[string]any{"type": "text", "fields": map[string]any{"keyword": map[string]any{"type": "keyword"}}},
				"description": map[string]any{"type": "text"},
				"category":    map[string]any{"type": "keyword"},
				"brand":       map[string]any{"type": "keyword"},
				"price":       map[string]any{"type": "float"},
				"currency":    map[string]any{"type": "keyword"},
				"in_stock":    map[string]any{"type": "boolean"},
				"quantity":    map[string]any{"type": "integer"},
				"rating":      map[string]any{"type": "float"},
				"tags":        map[string]any{"type": "keyword"},
				"created_at":  map[string]any{"type": "date"},
			},
		},
	}); err != nil {
		return fmt.Errorf("create products: %w", err)
	}

	now := time.Now().UTC()
	products := []map[string]any{
		{"sku": "ES-WID-001", "name": "Elastic Widget", "description": "Compact search appliance for demos and workshops.", "category": "hardware", "brand": "elastic", "price": 49.99, "currency": "USD", "in_stock": true, "quantity": 120, "rating": 4.6, "tags": []string{"search", "demo", "hardware"}, "created_at": now.AddDate(0, -3, 0).Format(time.RFC3339)},
		{"sku": "OS-GAD-002", "name": "OpenSearch Gadget", "description": "Open-source analytics gadget with dashboards baked in.", "category": "hardware", "brand": "opensearch", "price": 59.50, "currency": "USD", "in_stock": true, "quantity": 85, "rating": 4.4, "tags": []string{"search", "oss", "analytics"}, "created_at": now.AddDate(0, -2, -5).Format(time.RFC3339)},
		{"sku": "KB-MUG-003", "name": "Kibana Mug", "description": "Ceramic mug. Visualize your coffee intake.", "category": "merch", "brand": "elastic", "price": 14.00, "currency": "USD", "in_stock": true, "quantity": 400, "rating": 4.9, "tags": []string{"merch", "kibana"}, "created_at": now.AddDate(0, -1, -2).Format(time.RFC3339)},
		{"sku": "DB-TEE-004", "name": "Dashboards Tee", "description": "Soft cotton tee with OpenSearch Dashboards art.", "category": "merch", "brand": "opensearch", "price": 28.00, "currency": "USD", "in_stock": true, "quantity": 210, "rating": 4.5, "tags": []string{"merch", "oss"}, "created_at": now.AddDate(0, -1, 0).Format(time.RFC3339)},
		{"sku": "ES-STK-005", "name": "Elastic Stack Sticker Pack", "description": "10 vinyl stickers: ES, Kibana, Beats, Logstash.", "category": "merch", "brand": "elastic", "price": 8.50, "currency": "USD", "in_stock": true, "quantity": 999, "rating": 4.8, "tags": []string{"merch", "stickers"}, "created_at": now.AddDate(0, 0, -20).Format(time.RFC3339)},
		{"sku": "OS-NBK-006", "name": "OpenSearch Notebook", "description": "Dot-grid notebook for query plans and war stories.", "category": "merch", "brand": "opensearch", "price": 12.00, "currency": "USD", "in_stock": false, "quantity": 0, "rating": 4.2, "tags": []string{"merch", "oss"}, "created_at": now.AddDate(0, 0, -18).Format(time.RFC3339)},
		{"sku": "ES-CRT-007", "name": "Cluster Control Board", "description": "USB desk toy that blinks green/yellow/red with cluster health.", "category": "hardware", "brand": "elastic", "price": 79.00, "currency": "USD", "in_stock": true, "quantity": 40, "rating": 4.1, "tags": []string{"hardware", "ops", "demo"}, "created_at": now.AddDate(0, 0, -14).Format(time.RFC3339)},
		{"sku": "OS-HAT-008", "name": "Search Ops Cap", "description": "Adjustable cap. Embroidery: match_all {}.", "category": "merch", "brand": "opensearch", "price": 22.00, "currency": "USD", "in_stock": true, "quantity": 150, "rating": 4.3, "tags": []string{"merch", "ops"}, "created_at": now.AddDate(0, 0, -10).Format(time.RFC3339)},
		{"sku": "ES-BOK-009", "name": "Query DSL Field Guide", "description": "Printed guide to bool, nested, and aggs patterns.", "category": "books", "brand": "elastic", "price": 34.99, "currency": "USD", "in_stock": true, "quantity": 75, "rating": 4.7, "tags": []string{"books", "search"}, "created_at": now.AddDate(0, 0, -7).Format(time.RFC3339)},
		{"sku": "OS-BOK-010", "name": "Observability Playbook", "description": "Logs, metrics, and traces for OpenSearch users.", "category": "books", "brand": "opensearch", "price": 39.00, "currency": "USD", "in_stock": true, "quantity": 60, "rating": 4.6, "tags": []string{"books", "observability", "oss"}, "created_at": now.AddDate(0, 0, -5).Format(time.RFC3339)},
		{"sku": "ES-LIC-011", "name": "Trial License Key (demo)", "description": "Non-functional demo SKU for order flows.", "category": "software", "brand": "elastic", "price": 0.00, "currency": "USD", "in_stock": true, "quantity": 10000, "rating": 3.5, "tags": []string{"software", "demo"}, "created_at": now.AddDate(0, 0, -3).Format(time.RFC3339)},
		{"sku": "OS-SRV-012", "name": "Managed Cluster Credit", "description": "$50 credit for sandbox clusters.", "category": "software", "brand": "opensearch", "price": 50.00, "currency": "USD", "in_stock": true, "quantity": 500, "rating": 4.0, "tags": []string{"software", "cloud"}, "created_at": now.AddDate(0, 0, -1).Format(time.RFC3339)},
	}
	if err := bulkIndex(client, addr, "products", products); err != nil {
		return err
	}
	fmt.Fprintf(seedStdout, "  products:     %d docs\n", len(products))
	return nil
}

func seedCustomers(client *http.Client, addr string) error {
	if err := put(client, addr+"/customers", map[string]any{
		"settings": map[string]any{"number_of_shards": 1, "number_of_replicas": 0},
		"mappings": map[string]any{
			"properties": map[string]any{
				"customer_id": map[string]any{"type": "keyword"},
				"email":       map[string]any{"type": "keyword"},
				"name":        map[string]any{"type": "text", "fields": map[string]any{"keyword": map[string]any{"type": "keyword"}}},
				"company":     map[string]any{"type": "keyword"},
				"plan":        map[string]any{"type": "keyword"},
				"country":     map[string]any{"type": "keyword"},
				"city":        map[string]any{"type": "keyword"},
				"active":      map[string]any{"type": "boolean"},
				"ltv":         map[string]any{"type": "float"},
				"signup_at":   map[string]any{"type": "date"},
				"tags":        map[string]any{"type": "keyword"},
			},
		},
	}); err != nil {
		return fmt.Errorf("create customers: %w", err)
	}

	now := time.Now().UTC()
	customers := []map[string]any{
		{"customer_id": "cust_1001", "email": "alice@example.com", "name": "Alice Nguyen", "company": "SearchCo", "plan": "enterprise", "country": "US", "city": "Austin", "active": true, "ltv": 12400.50, "signup_at": now.AddDate(-2, -1, 0).Format(time.RFC3339), "tags": []string{"vip", "ops"}},
		{"customer_id": "cust_1002", "email": "bob@example.com", "name": "Bob Martinez", "company": "Logline", "plan": "pro", "country": "US", "city": "Seattle", "active": true, "ltv": 3200.00, "signup_at": now.AddDate(-1, -6, 0).Format(time.RFC3339), "tags": []string{"observability"}},
		{"customer_id": "cust_1003", "email": "carol@example.com", "name": "Carol Okonkwo", "company": "DataNest", "plan": "pro", "country": "GB", "city": "London", "active": true, "ltv": 5100.75, "signup_at": now.AddDate(-1, -2, 0).Format(time.RFC3339), "tags": []string{"search"}},
		{"customer_id": "cust_1004", "email": "dave@example.com", "name": "Dave Chen", "company": "Indie Labs", "plan": "free", "country": "CA", "city": "Toronto", "active": true, "ltv": 0, "signup_at": now.AddDate(0, -8, 0).Format(time.RFC3339), "tags": []string{"trial"}},
		{"customer_id": "cust_1005", "email": "eve@example.com", "name": "Eve Kowalski", "company": "RetailStack", "plan": "enterprise", "country": "DE", "city": "Berlin", "active": false, "ltv": 8900.00, "signup_at": now.AddDate(-3, 0, 0).Format(time.RFC3339), "tags": []string{"churned", "vip"}},
		{"customer_id": "cust_1006", "email": "frank@example.com", "name": "Frank Silva", "company": "Pipelines IO", "plan": "pro", "country": "BR", "city": "São Paulo", "active": true, "ltv": 2100.25, "signup_at": now.AddDate(0, -4, 0).Format(time.RFC3339), "tags": []string{"ingest"}},
		{"customer_id": "cust_1007", "email": "grace@example.com", "name": "Grace Park", "company": "Vectorly", "plan": "enterprise", "country": "KR", "city": "Seoul", "active": true, "ltv": 15600.00, "signup_at": now.AddDate(-2, -8, 0).Format(time.RFC3339), "tags": []string{"vip", "ml"}},
		{"customer_id": "cust_1008", "email": "henry@example.com", "name": "Henry Brooks", "company": "Nightwatch", "plan": "pro", "country": "AU", "city": "Sydney", "active": true, "ltv": 4400.00, "signup_at": now.AddDate(0, -11, 0).Format(time.RFC3339), "tags": []string{"security"}},
	}
	if err := bulkIndex(client, addr, "customers", customers); err != nil {
		return err
	}
	fmt.Fprintf(seedStdout, "  customers:    %d docs\n", len(customers))
	return nil
}

func seedOrders(client *http.Client, addr string) error {
	if err := put(client, addr+"/orders", map[string]any{
		"settings": map[string]any{"number_of_shards": 1, "number_of_replicas": 0},
		"mappings": map[string]any{
			"properties": map[string]any{
				"order_id":    map[string]any{"type": "keyword"},
				"customer_id": map[string]any{"type": "keyword"},
				"customer":    map[string]any{"type": "keyword"},
				"status":      map[string]any{"type": "keyword"},
				"currency":    map[string]any{"type": "keyword"},
				"total":       map[string]any{"type": "float"},
				"items": map[string]any{
					"type": "nested",
					"properties": map[string]any{
						"sku":      map[string]any{"type": "keyword"},
						"name":     map[string]any{"type": "text"},
						"qty":      map[string]any{"type": "integer"},
						"unit_price": map[string]any{"type": "float"},
					},
				},
				"shipping_city":    map[string]any{"type": "keyword"},
				"shipping_country": map[string]any{"type": "keyword"},
				"created_at":       map[string]any{"type": "date"},
				"updated_at":       map[string]any{"type": "date"},
			},
		},
	}); err != nil {
		return fmt.Errorf("create orders: %w", err)
	}

	now := time.Now().UTC()
	orders := []map[string]any{
		{
			"order_id": "ORD-1001", "customer_id": "cust_1001", "customer": "alice@example.com", "status": "shipped", "currency": "USD", "total": 77.99,
			"items": []map[string]any{
				{"sku": "ES-WID-001", "name": "Elastic Widget", "qty": 1, "unit_price": 49.99},
				{"sku": "KB-MUG-003", "name": "Kibana Mug", "qty": 2, "unit_price": 14.00},
			},
			"shipping_city": "Austin", "shipping_country": "US",
			"created_at": now.Add(-72 * time.Hour).Format(time.RFC3339),
			"updated_at": now.Add(-24 * time.Hour).Format(time.RFC3339),
		},
		{
			"order_id": "ORD-1002", "customer_id": "cust_1002", "customer": "bob@example.com", "status": "pending", "currency": "USD", "total": 28.00,
			"items": []map[string]any{
				{"sku": "DB-TEE-004", "name": "Dashboards Tee", "qty": 1, "unit_price": 28.00},
			},
			"shipping_city": "Seattle", "shipping_country": "US",
			"created_at": now.Add(-6 * time.Hour).Format(time.RFC3339),
			"updated_at": now.Add(-6 * time.Hour).Format(time.RFC3339),
		},
		{
			"order_id": "ORD-1003", "customer_id": "cust_1003", "customer": "carol@example.com", "status": "delivered", "currency": "USD", "total": 99.50,
			"items": []map[string]any{
				{"sku": "OS-GAD-002", "name": "OpenSearch Gadget", "qty": 1, "unit_price": 59.50},
				{"sku": "ES-BOK-009", "name": "Query DSL Field Guide", "qty": 1, "unit_price": 34.99},
			},
			"shipping_city": "London", "shipping_country": "GB",
			"created_at": now.Add(-240 * time.Hour).Format(time.RFC3339),
			"updated_at": now.Add(-120 * time.Hour).Format(time.RFC3339),
		},
		{
			"order_id": "ORD-1004", "customer_id": "cust_1007", "customer": "grace@example.com", "status": "processing", "currency": "USD", "total": 129.00,
			"items": []map[string]any{
				{"sku": "ES-CRT-007", "name": "Cluster Control Board", "qty": 1, "unit_price": 79.00},
				{"sku": "OS-SRV-012", "name": "Managed Cluster Credit", "qty": 1, "unit_price": 50.00},
			},
			"shipping_city": "Seoul", "shipping_country": "KR",
			"created_at": now.Add(-18 * time.Hour).Format(time.RFC3339),
			"updated_at": now.Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			"order_id": "ORD-1005", "customer_id": "cust_1004", "customer": "dave@example.com", "status": "cancelled", "currency": "USD", "total": 12.00,
			"items": []map[string]any{
				{"sku": "OS-NBK-006", "name": "OpenSearch Notebook", "qty": 1, "unit_price": 12.00},
			},
			"shipping_city": "Toronto", "shipping_country": "CA",
			"created_at": now.Add(-48 * time.Hour).Format(time.RFC3339),
			"updated_at": now.Add(-40 * time.Hour).Format(time.RFC3339),
		},
		{
			"order_id": "ORD-1006", "customer_id": "cust_1008", "customer": "henry@example.com", "status": "shipped", "currency": "USD", "total": 61.00,
			"items": []map[string]any{
				{"sku": "OS-HAT-008", "name": "Search Ops Cap", "qty": 1, "unit_price": 22.00},
				{"sku": "OS-BOK-010", "name": "Observability Playbook", "qty": 1, "unit_price": 39.00},
			},
			"shipping_city": "Sydney", "shipping_country": "AU",
			"created_at": now.Add(-96 * time.Hour).Format(time.RFC3339),
			"updated_at": now.Add(-30 * time.Hour).Format(time.RFC3339),
		},
		{
			"order_id": "ORD-1007", "customer_id": "cust_1001", "customer": "alice@example.com", "status": "delivered", "currency": "USD", "total": 8.50,
			"items": []map[string]any{
				{"sku": "ES-STK-005", "name": "Elastic Stack Sticker Pack", "qty": 1, "unit_price": 8.50},
			},
			"shipping_city": "Austin", "shipping_country": "US",
			"created_at": now.Add(-360 * time.Hour).Format(time.RFC3339),
			"updated_at": now.Add(-300 * time.Hour).Format(time.RFC3339),
		},
		{
			"order_id": "ORD-1008", "customer_id": "cust_1006", "customer": "frank@example.com", "status": "pending", "currency": "USD", "total": 50.00,
			"items": []map[string]any{
				{"sku": "OS-SRV-012", "name": "Managed Cluster Credit", "qty": 1, "unit_price": 50.00},
			},
			"shipping_city": "São Paulo", "shipping_country": "BR",
			"created_at": now.Add(-3 * time.Hour).Format(time.RFC3339),
			"updated_at": now.Add(-3 * time.Hour).Format(time.RFC3339),
		},
	}
	if err := bulkIndex(client, addr, "orders", orders); err != nil {
		return err
	}
	fmt.Fprintf(seedStdout, "  orders:       %d docs\n", len(orders))
	return nil
}

func seedLogs(client *http.Client, addr string) error {
	if err := put(client, addr+"/logs-app", map[string]any{
		"settings": map[string]any{"number_of_shards": 1, "number_of_replicas": 0},
		"mappings": map[string]any{
			"properties": map[string]any{
				"@timestamp": map[string]any{"type": "date"},
				"level":      map[string]any{"type": "keyword"},
				"service":    map[string]any{"type": "keyword"},
				"host":       map[string]any{"type": "keyword"},
				"env":        map[string]any{"type": "keyword"},
				"trace_id":   map[string]any{"type": "keyword"},
				"message":    map[string]any{"type": "text"},
				"http": map[string]any{
					"properties": map[string]any{
						"method":      map[string]any{"type": "keyword"},
						"path":        map[string]any{"type": "keyword"},
						"status_code": map[string]any{"type": "integer"},
						"duration_ms": map[string]any{"type": "float"},
					},
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("create logs-app: %w", err)
	}

	now := time.Now().UTC()
	services := []string{"api-gateway", "search-service", "billing", "auth", "indexer"}
	hosts := []string{"es-tui-node-1", "es-tui-node-2", "es-tui-node-3"}
	levels := []string{"debug", "info", "info", "info", "warn", "error"}
	paths := []string{"/v1/search", "/v1/orders", "/v1/customers", "/health", "/v1/index", "/v1/auth/login"}
	methods := []string{"GET", "GET", "POST", "GET", "PUT", "POST"}

	var docs []map[string]any
	for i := 0; i < 80; i++ {
		level := levels[i%len(levels)]
		svc := services[i%len(services)]
		status := 200
		msg := fmt.Sprintf("%s handled request ok", svc)
		if level == "warn" {
			status = 429
			msg = fmt.Sprintf("%s rate limit approaching", svc)
		}
		if level == "error" {
			status = []int{500, 502, 503}[i%3]
			msg = fmt.Sprintf("%s upstream failure status=%d", svc, status)
		}
		docs = append(docs, map[string]any{
			"@timestamp": now.Add(-time.Duration(i) * 90 * time.Second).Format(time.RFC3339),
			"level":      level,
			"service":    svc,
			"host":       hosts[i%len(hosts)],
			"env":        "demo",
			"trace_id":   fmt.Sprintf("trc-%06d", 100000+i),
			"message":    msg,
			"http": map[string]any{
				"method":      methods[i%len(methods)],
				"path":        paths[i%len(paths)],
				"status_code": status,
				"duration_ms": float64(5+i%120) + 0.25,
			},
		})
	}
	if err := bulkIndex(client, addr, "logs-app", docs); err != nil {
		return err
	}
	fmt.Fprintf(seedStdout, "  logs-app:     %d docs\n", len(docs))
	return nil
}

func seedMetrics(client *http.Client, addr string) error {
	if err := put(client, addr+"/metrics-host", map[string]any{
		"settings": map[string]any{"number_of_shards": 1, "number_of_replicas": 0},
		"mappings": map[string]any{
			"properties": map[string]any{
				"@timestamp":   map[string]any{"type": "date"},
				"host":         map[string]any{"type": "keyword"},
				"cpu_pct":      map[string]any{"type": "float"},
				"mem_pct":      map[string]any{"type": "float"},
				"disk_pct":     map[string]any{"type": "float"},
				"heap_pct":     map[string]any{"type": "float"},
				"search_qps":   map[string]any{"type": "float"},
				"index_docs_s": map[string]any{"type": "float"},
			},
		},
	}); err != nil {
		return fmt.Errorf("create metrics-host: %w", err)
	}

	now := time.Now().UTC()
	hosts := []string{"es-tui-node-1", "es-tui-node-2", "es-tui-node-3"}
	var docs []map[string]any
	for i := 0; i < 45; i++ {
		host := hosts[i%len(hosts)]
		docs = append(docs, map[string]any{
			"@timestamp":   now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			"host":         host,
			"cpu_pct":      12.0 + float64((i*7)%60),
			"mem_pct":      40.0 + float64((i*3)%45),
			"disk_pct":     55.0 + float64(i%20),
			"heap_pct":     30.0 + float64((i*5)%50),
			"search_qps":   100.0 + float64((i*13)%400),
			"index_docs_s": 10.0 + float64((i*3)%80),
		})
	}
	if err := bulkIndex(client, addr, "metrics-host", docs); err != nil {
		return err
	}
	fmt.Fprintf(seedStdout, "  metrics-host: %d docs\n", len(docs))
	return nil
}

func seedEvents(client *http.Client, addr string) error {
	if err := put(client, addr+"/events", map[string]any{
		"settings": map[string]any{"number_of_shards": 1, "number_of_replicas": 0},
		"mappings": map[string]any{
			"properties": map[string]any{
				"event_id":  map[string]any{"type": "keyword"},
				"type":      map[string]any{"type": "keyword"},
				"actor":     map[string]any{"type": "keyword"},
				"target":    map[string]any{"type": "keyword"},
				"payload":   map[string]any{"type": "object", "enabled": true},
				"timestamp": map[string]any{"type": "date"},
			},
		},
	}); err != nil {
		return fmt.Errorf("create events: %w", err)
	}

	now := time.Now().UTC()
	events := []map[string]any{
		{"event_id": "evt-1", "type": "user.login", "actor": "alice@example.com", "target": "auth", "payload": map[string]any{"ip": "203.0.113.10", "method": "password"}, "timestamp": now.Add(-5 * time.Hour).Format(time.RFC3339)},
		{"event_id": "evt-2", "type": "index.create", "actor": "system", "target": "products", "payload": map[string]any{"shards": 1, "replicas": 0}, "timestamp": now.Add(-4 * time.Hour).Format(time.RFC3339)},
		{"event_id": "evt-3", "type": "order.placed", "actor": "bob@example.com", "target": "ORD-1002", "payload": map[string]any{"total": 28.00, "items": 1}, "timestamp": now.Add(-6 * time.Hour).Format(time.RFC3339)},
		{"event_id": "evt-4", "type": "search.query", "actor": "search-service", "target": "products", "payload": map[string]any{"q": "tags:merch", "took_ms": 12}, "timestamp": now.Add(-2 * time.Hour).Format(time.RFC3339)},
		{"event_id": "evt-5", "type": "user.logout", "actor": "alice@example.com", "target": "auth", "payload": map[string]any{"session_min": 42}, "timestamp": now.Add(-90 * time.Minute).Format(time.RFC3339)},
		{"event_id": "evt-6", "type": "alert.fired", "actor": "monitor", "target": "cluster", "payload": map[string]any{"rule": "heap_high", "value": 82.5}, "timestamp": now.Add(-45 * time.Minute).Format(time.RFC3339)},
		{"event_id": "evt-7", "type": "order.shipped", "actor": "system", "target": "ORD-1001", "payload": map[string]any{"carrier": "demo-ship"}, "timestamp": now.Add(-24 * time.Hour).Format(time.RFC3339)},
		{"event_id": "evt-8", "type": "doc.index", "actor": "indexer", "target": "logs-app", "payload": map[string]any{"count": 80}, "timestamp": now.Add(-10 * time.Minute).Format(time.RFC3339)},
		{"event_id": "evt-9", "type": "user.login", "actor": "grace@example.com", "target": "auth", "payload": map[string]any{"ip": "198.51.100.22", "method": "sso"}, "timestamp": now.Add(-8 * time.Minute).Format(time.RFC3339)},
		{"event_id": "evt-10", "type": "search.query", "actor": "search-service", "target": "orders", "payload": map[string]any{"q": "status:pending", "took_ms": 8}, "timestamp": now.Add(-3 * time.Minute).Format(time.RFC3339)},
	}
	if err := bulkIndex(client, addr, "events", events); err != nil {
		return err
	}
	fmt.Fprintf(seedStdout, "  events:       %d docs\n", len(events))
	return nil
}

func bulkIndex(client *http.Client, addr, index string, docs []map[string]any) error {
	var buf bytes.Buffer
	for i, doc := range docs {
		meta := map[string]any{"index": map[string]any{"_index": index, "_id": fmt.Sprintf("%d", i+1)}}
		mb, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		db, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		buf.Write(mb)
		buf.WriteByte('\n')
		buf.Write(db)
		buf.WriteByte('\n')
	}
	req, err := http.NewRequest(http.MethodPost, addr+"/_bulk", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
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
		return fmt.Errorf("bulk HTTP %d: %s", resp.StatusCode, truncate(string(data), 300))
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err == nil {
		if errors, _ := result["errors"].(bool); errors {
			return fmt.Errorf("bulk had item errors: %s", truncate(string(data), 400))
		}
	}
	return nil
}

func put(client *http.Client, url string, body any) error {
	return doJSON(client, http.MethodPut, url, body)
}

func post(client *http.Client, url string, body any) error {
	return doJSON(client, http.MethodPost, url, body)
}

func del(client *http.Client, url string) error {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func doJSON(client *http.Client, method, url string, body any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
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
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(data), 300))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
