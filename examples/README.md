# Demo clusters

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

Both:

```bash
make docker-up
make docker-seed
make docker-down
```
