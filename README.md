# NASAOJ v3 Judge backend / worker / toolchains

[Wiki](https://gitlab.it.cs.nycu.edu.tw/groups/nasa/nasaoj-v3/-/wikis/home)

## Project Overview

NASAOJ Judge is a distributed online judge system for automated problem
submission and grading. It evaluates student submissions across multiple
problem stages with dependency management and distributed worker processing.

**Language**: Go 1.26.1 \
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
   - PostgreSQL for persistent state, including the durable submission queue
   - Redis for resource-group mutexes, worker-status/lease bookkeeping,
     score/user-args caching, and API tokens (no longer used for job queuing)
   - MinIO for object storage (problem/tool tars, per-job scripts, stdout/stderr)
   - Background crons reclaim work from dead workers (see below)

2. **Worker** (`cmds/serve/worker.go`) - Job execution service
   - Claims a whole **submission** at a time (not individual jobs) and
     resolves that submission's stage DAG itself, in-memory
   - For each ready testcase, asks the backend to marshal the job on demand,
     downloads its tar, and executes it (bash or, if `--container-runtime`
     is set, a CRI container), then reports the result
   - Runs a bounded pool of concurrent submissions, each running a bounded
     pool of concurrent jobs; sends a periodic heartbeat and honors
     backend-initiated cancellation

### Data Flow

Submission-level scheduling: the backend hands a worker one whole
problem/tool submission; the worker owns per-stage dependency resolution and
pulls each job's script/tar lazily as it becomes runnable.

```
POST /api/submit
  → validate problem/tool + namespace membership
  → atomically insert the Submission, every Job row (Pending), and a
    SubmissionQueueItem, all in one Postgres transaction

Worker polls GET /schedule/submission?tags=...
  → SELECT ... FOR UPDATE SKIP LOCKED claims one queued submission
  → the submission's jobs flip Pending -> Blocked

Worker resolves the stage DAG locally; for each testcase whose stage
becomes ready:
  → POST /schedule/job/marshal
      (job Blocked -> Running; backend renders scripts/args from the
       problem/tool YAML, writes the ref+job tars to MinIO, returns
       presigned download URLs)
  → worker downloads the tar and runs it, honoring timeout/grace period
  → PUT /schedule/result
      (backend persists stdout/stderr + terminal status; on failure it
       marks the whole downstream branch Skipped)

Once every job of a submission is terminal, the backend drops it from the
queue and scores are recalculated. Meanwhile the worker POSTs
/schedule/heartbeat every few seconds to refresh its queue lease, report
which submissions it's still running, and learn which of them the backend
wants cancelled; a cron reclaims submissions leased to a worker that stops
heartbeating.
```

Running jobs can also call back into the worker's own local API
(`PUT /args`, `PUT /score`), authenticated with a short-lived, worker-signed
job token; the worker verifies the token and forwards the request to the
backend on the job's behalf.

### Key Directories

- `backend/handlers/` - HTTP handlers (`api/` client-facing REST, `auth/`
  login, `schedule/` the worker-facing claim/marshal/result/heartbeat API)
- `backend/middlewares/` - HTTP middlewares (auth, namespace resolution,
  errors, access logs)
- `backend/workflows/` - small backend-side orchestration: `completeSubmission`
  (drop a submission from the queue once all its jobs are terminal),
  `skipJobs` (mark a failed stage's downstream jobs Skipped), `registerUser`
- `backend/crons/` - `checkWorkerStatus`, `cleanOrphanedJobs`,
  `reclaimSubmissions` (returns submissions leased to dead workers to the queue)
- `models/` - Bun ORM models (job, submission, submissionQueue, problem,
  stage, tool, namespace)
- `interfaces/` - Storage interfaces (database, jobQueue, submissionQueueStorage,
  problemStorage, namespaceStorage, ...)
- `implements/` - PostgreSQL (submissions/jobs/queue, atomic per submission),
  Redis (resource-group locks, worker status, caching, tokens), MinIO
- `worker/workflows/` - `runSubmission` (claim + in-memory stage DAG
  resolution), `runJob`/`runShellJob`/`runContainerJob` (execution),
  `heartbeat`, `cancelRegistry`, `jwt` (job token signing/verification)
- `worker/handlers/api/` - the worker-local API a running job calls
  (`PUT /args`, `PUT /score`), forwarded to the backend
- `worker/cri/` - CRI (container runtime) client for container-mode jobs
- `parsers/` - YAML parsing for problems/stages/testcases
- `utils/` - Helpers (tar handling, score calculation, file loading)
- `cache/` - small cache abstraction used for score/user-args invalidation
- `migrations/` - Database migrations

### Important Patterns

**Job lifecycle** (`models/job.go`): `Pending` (submission queued) ->
`Blocked` (submission claimed by a worker) -> `Running` (job marshalled) ->
terminal: `Success` / `Failed` / `Timeout` / `Cancelled` / `Skipped`
(upstream stage failed) / `Missing` (orphaned, worker died mid-job). A dead
worker's `Blocked` jobs are reset to `Pending` by the reclaim cron so another
worker can pick the submission back up.

**Stage Dependencies**: Stages define `before`/`after` arrays. Parsing converts
`after` to `before` references. Circular dependencies are validated during
problem parsing. Dependency resolution itself now happens on the worker, not
the backend - the backend only stores the DAG and skips downstream jobs when
a stage fails.

**Content System**: Problems/tools stored as YAML. Content files referenced by
SHA1. Ref tars (problem/tool content, written once per SHA1) and per-job tars
(rendered args baked in) are built lazily by `POST /schedule/job/marshal` and
distributed to workers as presigned MinIO URLs.

**Namespaces**: Problems, tools, submissions, and jobs are scoped to a
namespace (see `backend/middlewares/checkNamespace.go`); a submission is
created in the namespace its target problem/tool belongs to, and that
namespace rides along on the `SubmissionQueueItem`/`Job` rows a worker sees.

## CLI Commands

```bash
judge serve backend/worker  # Serve backend/worker server
judge migrate ...           # Database (Postgres) migration
judge admin ...             # Admin CLIs
```
