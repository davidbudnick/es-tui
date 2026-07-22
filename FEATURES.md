# es-tui feature roadmap

Goal: the best keyboard-first TUI for Elasticsearch **and** OpenSearch.

## Status

| Mark | Meaning |
| --- | --- |
| ✅ | Implemented |
| 🔨 | Partial |
| ⬜ | Planned (next) |

### 1. Connection & profiles

| Feature | Status |
| --- | --- |
| Multiple saved profiles + groups | ✅ |
| Basic auth + API key + **Bearer token** | ✅ |
| TLS (CA / cert / skip-verify) | ✅ |
| Auto-detect ES vs OpenSearch + version | ✅ |
| CLI quick-connect (+ `--bearer`, `--read-only`) | ✅ |
| **Read-only mode** (blocks mutations) | ✅ |
| Health / test on connect | ✅ |
| Secrets stripped from disk (password, api key, bearer) | ✅ |
| AWS SigV4 / SSH tunnel | ⬜ |

### 2. Cluster overview & monitoring

| Feature | Status |
| --- | --- |
| Cluster health + status bar chip | ✅ |
| Nodes table (roles, heap, CPU, disk) | ✅ |
| Live metrics + auto-refresh + sparkline | ✅ |
| Disk **allocation** table | ✅ |
| Tasks list + cancel | ✅ |
| Plugins list | ✅ |
| Cluster settings (JSON) | ✅ |
| Advanced JVM/GC charts | 🔨 |

### 3. Indices

| Feature | Status |
| --- | --- |
| Split-pane list + preview, blue selection | ✅ |
| Create / delete / open / close / refresh / force-merge | ✅ |
| Settings + mappings JSON | ✅ |
| Aliases + templates | ✅ |
| **Data streams** | ✅ |
| **Reindex** (async task) | ✅ |
| **Count** API | ✅ |
| Filter pattern | ✅ |

### 4. Documents & search

| Feature | Status |
| --- | --- |
| Split-pane browser + JSON highlight | ✅ |
| CRUD + bulk delete-by-query | ✅ |
| Query-string + raw JSON DSL | ✅ |
| Split-pane search + field preview | ✅ |
| Pagination n/p | ✅ |
| Query history (session) + **saved queries (disk)** | ✅ |
| **Explain** API | ✅ |
| **Export** NDJSON | ✅ |
| Copy JSON (`y`) | ✅ |
| Aggregations builder | ⬜ |

### 5. Nodes / shards / cat

| Feature | Status |
| --- | --- |
| Cat nodes / shards / aliases / cat explorer | ✅ |
| Allocation | ✅ |
| Reroute wizard | ⬜ |

### 6. Advanced

| Feature | Status |
| --- | --- |
| Snapshots (list by repo) | ✅ |
| Tasks + cancel | ✅ |
| Plugins | ✅ |
| Cluster settings view | ✅ |
| ILM write policies | ⬜ |
| Snapshot create/restore UI | ⬜ |

### 7. UX

| Feature | Status |
| --- | --- |
| Vim nav, help, confirmations | ✅ |
| Split panes (indices/docs/search) | ✅ |
| **Command palette (`:`)** | ✅ |
| Clipboard | ✅ |
| Status: Connected · flavor · RO · health | ✅ |
| Themes | ⬜ |

---

## Keyboard (highlights)

| Context | Keys |
| --- | --- |
| Global (connected) | `:` palette · `?` help · `q` back |
| Indices | `/` search · `O/X` open/close · `u` refresh · `M` merge · `I` reindex · `V` alloc · `W` tasks · `E` data streams · `U` settings · `Z` snapshots · `Y` saved queries · `Q` export · `#` count |
| Search | `enter` run · `j/k` hits · `n/p` page · `y` copy · `S` save query · `x` explain · `#` count |
| Documents | `/` search · `f` inline filter · `y` copy · `n/p` page |
| Palette | type to filter · `enter` run |

---

Still planned (SigV4, SSH, ILM editors, snapshot create/restore, themes) — solid daily admin surface is in place.
