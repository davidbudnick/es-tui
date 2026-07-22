# Demo clusters

Rich sample data for exercising es-tui (indices, search, docs, aliases, logs).

## Indices seeded

| Index | Docs | Purpose |
| --- | ---: | --- |
| `products` | 12 | Catalog with categories, tags, stock, ratings |
| `customers` | 8 | Accounts with plan, country, LTV |
| `orders` | 8 | Nested line items + shipping + status |
| `logs-app` | 80 | App logs with http method/path/status/trace_id |
| `metrics-host` | 45 | Host CPU/mem/heap/qps time series |
| `events` | 10 | Product analytics events |

Aliases: `shop`, `catalog` → `products`

## Elasticsearch (port 9200)

```bash
make docker-up-es
make docker-seed-es
./bin/es-tui --host localhost --port 9200
```

## OpenSearch (port 9201)

```bash
make docker-up-os
make docker-seed-os
./bin/es-tui --host localhost --port 9201 --flavor opensearch
```

## Both

```bash
make docker-up
make docker-seed
make docker-down
```

## Manual seed

```bash
go run ./examples/seed -addr http://localhost:9200 -flush
go run ./examples/seed -addr http://localhost:9201 -flush
```

### Useful searches in the TUI

- `products` query: `tags:merch` or `category:hardware`
- `orders` query: `status:pending`
- `logs-app` query: `level:error` or `service:auth`
- `customers` query: `plan:enterprise`
