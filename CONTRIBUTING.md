# Contributing

Developer-facing documentation: building, testing, and the codebase's
architecture. For what the project is and how to run it, see
[README.md](README.md).

**Language**: Go 1.26.1 \
**Module**: `gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge`

## Development Commands

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
      (backend persists the terminal status; on failure it marks the whole
       downstream branch Skipped)

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

Stdout/stderr and any testcase-declared artifacts are uploaded by the
worker directly to object storage via presigned PUT URLs the backend issues
at marshal time - the backend never sees those bytes, only the ref/job tar
content and the final job status.

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
  stage, tool, namespace, token)
- `interfaces/` - Storage interfaces (database, jobQueue, submissionQueueStorage,
  problemStorage, namespaceStorage, tokenStorage, ...)
- `implements/` - PostgreSQL (submissions/jobs/queue/tokens, atomic per
  submission), Redis (resource-group locks, worker status, caching - optional,
  see `--single-node`), Local (in-process stand-in for Redis), MinIO
- `worker/workflows/` - `runSubmission` (claim + in-memory stage DAG
  resolution), `runJob`/`runShellJob`/`runContainerJob` (execution),
  `artifacts` (per-job cleanup, artifact collection/seeding across dependent
  stages), `heartbeat`, `cancelRegistry`, `jwt` (job token signing/verification)
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

**Job working directories & artifacts**: A job's working directory is
deleted the moment that job finishes - only files a testcase explicitly
declares via `Testcase.Artifacts` (glob patterns) survive, archived and
uploaded straight to object storage. Kept artifacts stay visible on disk to
any testcase in a stage that transitively depends on the producing stage,
for the rest of the submission's run (`worker/workflows/artifacts.go`
computes the exact ancestor-stage set from `Stage.Before`), but never to
unrelated concurrent branches. Retrieval is admin-only (`GET
/api/admin/artifacts`, or `/api/admin/artifacts/url` for a presigned
direct-download link), the same access level as stderr.

**Tokens & Redis**: Static bearer tokens (admin/worker/user-scoped) are
stored in Postgres (`models.Token`), not Redis. Redis is optional
(`--single-node` / `SINGLE_NODE=true` runs without it, using an in-process
store for resource-group locks and worker status instead); when configured,
it's used only as a cache layer in front of Postgres.
