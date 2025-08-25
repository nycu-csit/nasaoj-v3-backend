# Judge

## How to run

### Service

Run `docker compose up`. \
Rebuild with `make docker`.

#### Bootstrap

1. Fill in env in `backend/.env`.
2. Run
```sh
docker compose exec backend /judge migrate init
docker compose exec backend /judge migrate up
```

### CLI

Make sure `~/go/bin/` is in your `PATH`. \
Build and install with `make install`.
