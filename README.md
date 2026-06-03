# Cosmo

![License](https://img.shields.io/badge/license-MIT-blue)
![Version](https://img.shields.io/badge/version-v0.2.0-cyan)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8)

A real-time PostgreSQL internals dashboard for your terminal. Built in Go using Bubbletea and Lipgloss.

![startup](Wi.gif)

---

## What is Cosmo?

Most developers use PostgreSQL every day without actually knowing what's happening inside it. Cosmo fixes that.

It connects directly to your Postgres database and gives you a live view of everything — active queries, WAL activity, connection health, lock contention — all updating in real time, right in your terminal.

No browser. No external service. No configuration beyond a database URL. Just run it and watch your database breathe.

---

## Features

**DB Overview panel**
- Database name and PostgreSQL version
- Database size
- Active connections with visual progress bar
- Cache hit ratio with visual progress bar and health colors
- Server uptime
- Total transaction count (comma formatted)

**Active Queries panel**
- Live view of all running queries
- Query state (active, idle, idle in transaction)
- Duration — how long each query has been running
- Truncated query text

**WAL & MVCC panel**
- Current WAL LSN (Log Sequence Number)
- Real-time WAL write rate in MB/s with progress bar
- Dead tuples and live tuples
- Checkpoint count
- Last autovacuum time

**Locks & Waits panel**
- Active lock type and status (granted/waiting)
- Which table is locked
- PID of the process holding or waiting for the lock
- Query text

**General**
- Mission control boot sequence on startup
- Auto-refreshes every 2 seconds
- Health-based colors — green for healthy, amber for warning, red for critical
- Live clock in header
- TAB to switch between panels
- R to manually refresh
- Q to quit

---

## Demo

![Demo](demo.gif)

The GIF shows:
1. Cosmo booting up with the mission control startup animation
2. The full dashboard appearing with all 4 panels populated with live data
3. A `SELECT pg_sleep(20)` query being run in pgAdmin
4. The query appearing live in the Active Queries panel with duration counting up
5. The query finishing and disappearing from the panel

To record your own demo, use [ScreenToGif](https://www.screentogif.com/) on Windows.

---

## Architecture

```mermaid
flowchart TD
    A[PostgreSQL Database] --> B[pg_stat_activity]
    A --> C[pg_stat_database]
    A --> D[pg_stat_checkpointer]
    A --> E[pg_locks]

    B --> F[Cosmo — queries.go]
    C --> F
    D --> F
    E --> F

    F --> G[model.go — Bubbletea Model]
    G -->|every 2 seconds| H[render.go — Lipgloss Renderer]

    H --> I[DB Overview Panel]
    H --> J[Active Queries Panel]
    H --> K[WAL & MVCC Panel]
    H --> L[Locks & Waits Panel]
```

### Project Structure

```
cosmo/
├── main.go                          → entry point, wires everything together
├── config/
│   └── config.go                   → loads config from environment variables
├── internal/
│   ├── db/
│   │   └── queries.go              → all PostgreSQL queries and data structs
│   └── ui/
│       ├── model.go                → bubbletea model, tick loop, data fetching
│       ├── render.go               → lipgloss rendering for all 4 panels
│       ├── startup.go              → mission control boot sequence animation
│       └── panels/                 → reserved for future panel components
```

---

## How it works

Cosmo queries PostgreSQL system views directly — the same views that Postgres uses internally to track its own state. These are:

**pg_stat_activity** — every connection and what it's doing right now. This is how Cosmo shows active queries, their state, and how long they've been running.

**pg_stat_database** — database-level statistics like cache hits, block reads, transaction counts, and database size. The cache hit ratio is calculated from blks_hit and blks_read.

**pg_stat_checkpointer** — checkpoint statistics in PostgreSQL 17+. This is how Cosmo tracks how many checkpoints have happened.

**pg_locks** — every lock currently held or waited on. Cosmo joins this with pg_stat_activity and pg_class to show which table is involved and what query is holding the lock.

**WAL rate** is calculated by tracking the change in WAL LSN (Log Sequence Number) between refreshes. LSN is a byte position in the write-ahead log — by comparing two consecutive positions and dividing by elapsed time, Cosmo calculates the write throughput in MB/s.

Cosmo uses a connection pool via pgxpool to avoid connection conflicts when multiple queries run concurrently every 2 seconds.

---

## Setup

**Requirements**
- PostgreSQL 17+
- Go 1.26+

**Clone the repo**

```bash
git clone https://github.com/mujib77/cosmo
cd cosmo
```

**Create a .env file**

```
DATABASE_URL=postgres://user:password@localhost:5432/dbname
```

**Install dependencies**

```bash
go mod tidy
```

**Run**

```bash
go run main.go
```

---

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| TAB | Switch active panel |
| R | Manual refresh |
| Q | Quit |

---

## Built With

- [Go](https://golang.org/) — systems language
- [Bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [pgx](https://github.com/jackc/pgx) — PostgreSQL driver and connection pool

---

## License

MIT — see [LICENSE](LICENSE)

# test