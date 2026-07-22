# es-tui feature roadmap

Goal: the best keyboard-first TUI for Elasticsearch **and** OpenSearch — redis-tui UX density + ktsearch admin power + live monitoring.

## Status legend

| Mark | Meaning |
| --- | --- |
| ✅ | Implemented |
| 🔨 | Partial / improving |
| ⬜ | Planned |

---

## 1. Connection & profiles

| Feature | Status |
| --- | --- |
| Multiple saved profiles | ✅ |
| Connection groups | ✅ (config) |
| Basic auth + API key | ✅ |
| Bearer token | ⬜ |
| AWS SigV4 | ⬜ |
| TLS (CA / cert / skip-verify) | ✅ |
| SSH tunneling | ⬜ |
| Auto-detect ES vs OpenSearch + version | ✅ |
| CLI quick-connect flags | ✅ |
| Interactive profile switcher | ✅ |
| Read-only mode | ⬜ |
| Health / test on connect | ✅ |
| Passwords/API keys stripped from disk | ✅ |

## 2. Cluster overview & live monitoring

| Feature | Status |
| --- | --- |
| Cluster health (green/yellow/red) | ✅ |
| Node count / shard summary | ✅ |
| Live metrics dashboard + auto-refresh | 🔨 |
| Node list: roles, CPU, heap, disk | 🔨 |
| Search/indexing rates + latency | 🔨 |
| JVM / GC / network charts | ⬜ |
| ASCII charts | 🔨 |
| Color-coded health everywhere | 🔨 |

## 3. Indices management

| Feature | Status |
| --- | --- |
| Filterable index list + preview pane | ✅ |
| Columns: health, docs, size, shards, status | ✅ |
| Create / delete | ✅ |
| Open / close / refresh | ✅ |
| Force-merge | ✅ |
| Settings / mappings JSON view | ✅ |
| Aliases | ✅ |
| Templates | ✅ |
| Data streams | ⬜ |
| Reindex wizard | ⬜ |
| Fuzzy / filter | 🔨 |

## 4. Documents & search

| Feature | Status |
| --- | --- |
| Paginated document browser + preview | ✅ |
| JSON viewer + syntax highlight | ✅ |
| CRUD (get/index/delete) | ✅ |
| Bulk delete-by-query | ✅ |
| Query-string search | ✅ |
| Raw JSON Query DSL | ✅ |
| Split-pane search results + preview | ✅ |
| Saved queries / history | 🔨 history in-session |
| Pagination (from/size) | ✅ |
| Aggregations / explain / highlight | ⬜ |
| Export results | ⬜ |

## 5. Nodes, shards & allocation

| Feature | Status |
| --- | --- |
| Cat nodes / shards / aliases | ✅ |
| Cat API explorer | ✅ |
| Allocation view | ⬜ |
| Shard balance analysis | ⬜ |
| Reroute helpers | ⬜ |

## 6. Advanced operations

| Feature | Status |
| --- | --- |
| ILM policies | ⬜ |
| Snapshots | ⬜ |
| Task list / cancel | ⬜ |
| Dump / restore NDJSON | ⬜ |
| Cluster settings edit | ⬜ |
| Plugins list | ⬜ |

## 7. UX polish

| Feature | Status |
| --- | --- |
| Vim-style navigation | ✅ |
| Split panes (indices/docs/search) | 🔨 |
| Filtering in lists | 🔨 |
| Copy to clipboard | ✅ |
| Themes | ⬜ |
| Help (`?`) | ✅ |
| Confirmation on destructive actions | ✅ |
| Status bar connection + health | 🔨 |
| Multi-profile switch | ✅ |

---

## Phases

1. **MVP (current focus)** — connections, cluster health, index browser, documents, **great search**, CRUD  
2. **v1** — nodes/shards density, open/close/refresh/forcemerge, mappings polish, clipboard, pagination  
3. **v1.5** — query history/saved queries, reindex, bulk select, aliases write  
4. **Later** — ILM, snapshots, data streams, SigV4, SSH, themes  

This document is the product checklist. Implementation tracks these boxes, not a rewrite for its own sake.
