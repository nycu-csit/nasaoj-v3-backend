# NASAOJ v3 Judge backend / worker / toolchains

[Wiki](https://gitlab.it.cs.nycu.edu.tw/groups/nasa/nasaoj-v3/-/wikis/home)

## Project Overview

NASAOJ Judge is a distributed online judge system for automated problem
submission and grading. It evaluates student submissions across multiple
problem stages with dependency management and distributed worker processing.

## Quick Start

```bash
# Build swagger docs, install the judge CLI, and start the full Docker stack
# (PostgreSQL, Redis, MinIO, backend, worker)
make

# Initialize the database (after a fresh docker compose up)
make docker-init
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow,
architecture, and codebase layout.

## CLI Commands

The `judge` binary is both the backend/worker server and the admin/user
client for talking to a running instance.

```bash
judge serve backend/worker  # Serve the backend or worker server
judge migrate ...           # Database (Postgres) migration
judge admin ...             # Admin CLIs (import problems/tools, manage
                             # users, namespaces, tokens, submissions, ...)
```

Run `judge --help`, `judge admin --help`, etc. for the full command tree.
