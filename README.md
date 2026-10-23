# Data Sync

![CI](https://github.com/sahebifar/data-sync/actions/workflows/ci.yml/badge.svg)
![CD](https://github.com/sahebifar/data-sync/actions/workflows/cd.yml/badge.svg)

A full-stack data synchronization system that keeps **PostgreSQL** (source of truth) and **YugabyteDB** (distributed replica) in sync using the **Transactional Outbox** pattern, with a React dashboard for monitoring, comparison, and repair.

## Features

- **Write-time safety** — all writes go to PostgreSQL inside a single transaction that also inserts into `sync_outbox`. If the replica is down, nothing is lost.
- **Automatic synchronization** — a background worker polls `sync_outbox` and applies changes to YugabyteDB with exponential backoff retries.
- **Compare** — one endpoint diffs both databases and reports missing, extra, or mismatched users.
- **Repair** — re-sync missing users, fix mismatches, and (optionally) delete extra rows from the replica.
- **Observability** — full dashboard to inspect users in both databases, the outbox queue, and per-record errors.
- **Graceful degradation** — YugabyteDB can be offline; the API keeps accepting writes and the worker catches up once the replica returns.

## Architecture

```
Client (React) ──► Go Backend ──► PostgreSQL (source of truth)
                        │              │
                        │              ▼
                        │         sync_outbox
                        │              │
                        ▼              ▼
                   Outbox Worker ──► YugabyteDB (replica)
```

**Write flow:**

1. Client sends `POST /api/users/create`.
2. Backend opens a PostgreSQL transaction, inserts the user, and inserts a row into `sync_outbox` — then commits. Both writes are atomic.
3. A background worker picks up pending outbox rows every 2 seconds, applies them to YugabyteDB, and marks them `completed`.
4. Failed operations are retried with exponential backoff; after 10 attempts, they're marked `failed` and appear in the dashboard for manual retry.

**Why Outbox?** Writing to two databases without a distributed transaction is unsafe — one can succeed while the other fails. The outbox pattern turns this into a safe, eventually-consistent flow: the primary write is the source of truth, and the replica catches up asynchronously.

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.27 (`net/http`, `database/sql`) |
| Source of truth | PostgreSQL |
| Distributed replica | YugabyteDB (YSQL) |
| Frontend | React + TypeScript + Vite |
| Styling | Tailwind CSS v4 |
| CI/CD | GitHub Actions |
| Container | Docker (multi-stage) |

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/health` | Liveness check |
| GET | `/api/users` | List users from **PostgreSQL** |
| GET | `/api/users/yugabyte` | List users from **YugabyteDB** |
| POST | `/api/users/create` | Create user + enqueue sync |
| PUT | `/api/users/update?id=N` | Update user + enqueue sync |
| DELETE | `/api/users/delete?id=N` | Delete user + enqueue sync |
| GET | `/api/users/compare` | Diff both databases |
| POST | `/api/users/sync/missing` | Insert missing users into YugabyteDB |
| POST | `/api/users/repair` | Insert/update/delete to reconcile replica |
| GET | `/api/sync/outbox?status=X` | List outbox records |
| POST | `/api/sync/outbox/retry?id=N` | Retry a failed outbox record |

## Project Structure

```
.
├── .github/workflows/     CI + CD pipelines
├── database/
│   ├── database.go        PostgreSQL connection
│   ├── yugabyte.go        YugabyteDB connection
│   ├── migrate.go         Embedded SQL migration runner
│   ├── migrations/        Versioned SQL per database
│   ├── outbox.go          Enqueue within caller's transaction
│   └── outbox_worker.go   Background sync worker
├── handlers/              HTTP handlers + unit tests (sqlmock)
├── models/                Data models
├── routes/                Route registration
├── frontend/              React + TypeScript + Tailwind dashboard
├── Dockerfile
├── .env.example
└── main.go
```

## Running Locally

### Prerequisites

- Go 1.27+
- Node.js 20+
- PostgreSQL running locally
- YugabyteDB running (e.g., on a VM) and reachable from your host

### 1. Clone and configure

```bash
git clone https://github.com/sahebifar/data-sync.git
cd data-sync
cp .env.example .env
# Edit .env with your credentials
```

### 2. Start the backend

```bash
go run .
```

On first run, embedded migrations create the `users` and `sync_outbox` tables in PostgreSQL and the `users` table in YugabyteDB. Backend runs on `http://localhost:8080`.

### 3. Start the frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend runs on `http://localhost:5173`. Vite proxies `/api/*` to the backend, so no CORS configuration is needed.

## Testing

```bash
go test ./... -v
```

- `handlers/compare_logic_test.go` — pure comparison logic
- `handlers/create_user_test.go` — request validation + duplicate detection via `sqlmock`
- `models/user_test.go` — model field tests

No running database is required for the test suite.

## CI/CD

- **CI** — on every push: `go vet`, `go build`, `go test`.
- **CD** — on every `v*` tag: builds and publishes a Docker image to `ghcr.io/sahebifar/data-sync`.

## License

Built as a bachelor's thesis / portfolio project.