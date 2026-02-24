# NASAOJ v3 Judge backend / worker / toolchains

## Project Overview

NASAOJ Judge is a distributed online judge system for automated problem
submission and grading. It evaluates student submissions across multiple
problem stages with dependency management and distributed worker processing.

**Language**: Go 1.24
**Module**: `gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge`

## Common Commands

```bash
# Build swagger docs, install CLI, and restart Docker stack
make

# Run full Docker stack (PostgreSQL, Redis, MinIO, backend, worker)
make docker              # Rebuild and restart backend/worker

# Initialize database (after fresh docker compose up)
make docker-init

# Run tests
make test                # Unit + e2e tests
                         # DO NOT DIRECTLY RUN go test

# Full reset and test cycle
make retest              # docker-clean → docker → docker-init → test
```

## Architecture

### Core Services

1. **Backend** (`cmds/serve/backend.go`) - HTTP REST API server
   - PostgreSQL for persistent state
   - Redis for job queuing and caching
   - MinIO for object storage (problem files, test data)

2. **Worker** (`cmds/serve/worker.go`) - Job execution service
   - Fetches jobs from backend, executes bash scripts, reports results
   - Runs multiple threads (configurable)

### Data Flow

```
POST /api/submit → Submission created → RunProblem workflow
  → For each Stage (respecting dependencies):
    → Create Jobs with scripts + inputs → Enqueue to Redis
      → Worker fetches job → Executes with timeout → Reports result
        → Backend updates status → Triggers next stage → Calculates scores
```

### Key Directories

- `backend/handlers/` - HTTP handlers (api/, auth/, schedule/)
- `backend/middlewares/` - HTTP middlewares (auth, errors, access logs)
- `models/` - Bun ORM models (job, submission, problem, stage, tool)
- `interfaces/` - Storage interfaces (database, jobQueue, problemStorage)
- `implements/` - PostgreSQL, Redis, MinIO implementations
- `workflows/` - Orchestration (runProblem, runStage, runJob, handleJobDone)
- `worker/` - Worker client (fetchJob, putJob)
- `parsers/` - YAML parsing for problems/stages/testcases
- `utils/` - Helpers (tar handling, score calculation, file loading)
- `migrations/` - Database migrations

### Important Patterns

**Stage Dependencies**: Stages define `before`/`after` arrays. Parsing converts
`after` to `before` references. Circular dependencies are validated during
problem parsing.

**Content System**: Problems/tools stored as YAML. Content files referenced by
SHA1. Tar archives distribute jobs to workers.

## CLI Commands

```bash
judge serve backend/worker  # Serve backend/worker server
judge migrate ...           # Database (Postgres) migration
judge admin ...             # Admin CLIs
```
