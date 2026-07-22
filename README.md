# es-tui

[![CI](https://github.com/davidbudnick/es-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/davidbudnick/es-tui/actions/workflows/ci.yml)
[![Coverage: 100%](https://img.shields.io/badge/coverage-100%25-brightgreen)](https://github.com/davidbudnick/es-tui/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A feature-rich terminal UI for **Elasticsearch** and **OpenSearch**, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). Browse indices, search documents, and monitor cluster health without leaving the terminal.

The connection screen uses a multicolor palette inspired by the Elastic logo (pink, yellow, teal, blue, green).

## Quick Install

```bash
# Native install — recommended (macOS and Linux)
curl -fsSL https://raw.githubusercontent.com/davidbudnick/es-tui/main/install.sh | bash

# Go (requires Go 1.26+)
go install github.com/davidbudnick/es-tui@latest
```

## Features

### Browsing and Search

- **Index browser** with pattern filtering
- **Document browser** with query_string or full JSON Query DSL
- **Document detail** with JSON syntax highlighting
- **Search console** across one index or the whole cluster
- **Favorites and recent indices** for quick access

### Cluster and Index Ops

- **Cluster health** and **node list** (cat APIs)
- **Shards**, **aliases**, and **index templates**
- **Live metrics** — docs, store size, search latency, JVM heap, CPU
- **Create / delete indices**, open settings and mappings
- **Index / update / delete documents**, delete-by-query bulk delete
- **Cat API explorer** for ad-hoc `_cat/*` calls

### Connections

- **CLI quick connect** — `--host`, `--port`, `--user`, `--password`, `--api-key`, `--flavor`
- **Connection manager** with saved instances
- **TLS/SSL** support (client certs, CA, skip-verify)
- **Auto-detect** Elasticsearch vs OpenSearch (or force with `--flavor`)

## Usage

```bash
# Interactive connection manager
es-tui

# Quick connect to Elasticsearch (default port 9200)
es-tui --host localhost

# OpenSearch on 9201
es-tui --host localhost --port 9201 --flavor opensearch

# Basic auth + TLS
es-tui --host es.example.com --tls --user elastic --password secret

# API key
es-tui --host es.example.com --api-key "$ES_API_KEY" --tls

# Version / self-update
es-tui --version
es-tui --update
```

Press `?` inside the app for the full help screen.

### CLI Flags

| Flag | Short | Description | Default |
| --- | --- | --- | --- |
| `--host` | `-h` | Server hostname | |
| `--port` | `-p` | Server port | 9200 |
| `--password` | `-a` | Basic auth password | |
| `--user` | | Basic auth username | |
| `--api-key` | | API key auth | |
| `--name` | | Connection display name | `host:port` |
| `--flavor` | | `auto`, `elasticsearch`, `opensearch` | auto |
| `--tls` | | Enable TLS | false |
| `--tls-cert` | | Client certificate | |
| `--tls-key` | | Client key | |
| `--tls-ca` | | CA certificate | |
| `--tls-skip-verify` | | Skip TLS verify | false |
| `--version` | | Print version | |
| `--update` | | Self-update | |

### Local demo (Docker Desktop)

```bash
# Start Elasticsearch (:9200) and OpenSearch (:9201)
make docker-up

# Seed demo indices (products, orders, logs-demo)
make docker-seed

# Run the TUI
make run
# or: ./bin/es-tui --host localhost --port 9200
# or: ./bin/es-tui --host localhost --port 9201 --flavor opensearch

make docker-down
```

## Development

```bash
make build
make test
make test-cover-check   # enforces 100% per-function coverage
make lint
```

## License

MIT — see [LICENSE](LICENSE).
