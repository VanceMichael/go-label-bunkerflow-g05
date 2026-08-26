# BunkerFlow

BunkerFlow is a multi-tenant operations backend for ship-to-ship green methanol bunkering. It coordinates vessel certificates, terminal windows, fuel lot quality, safety permits, transfer execution, custody samples, invoices, audit events and retryable outbox delivery.

## Runtime

Go 1.26.0 is required and `GOTOOLCHAIN=local` is used by all Make targets. Set `DATABASE_URL` to a SQLite file and `HTTP_ADDR` to the listener address. Run `make run`, then check `GET /healthz` and `GET /readyz`.

Authentication uses seeded operator accounts in the migration. The demo credentials are only for local development: `planner@example.test` / `planner-pass` and `quality@example.test` / `quality-pass`.

## API areas

- `/api/v1/auth/login`, `/api/v1/auth/logout`
- `/api/v1/vessels`, `/api/v1/terminals`, `/api/v1/fuel-lots`
- `/api/v1/windows`, `/api/v1/bunkering`, `/api/v1/quality/samples`
- `/api/v1/invoices`, `/api/v1/audit/events`, `/api/v1/incidents`

All writes use a request context, a transaction where entities cross boundaries, an audit event and an outbox message. Workers honor cancellation, leases and retry limits.

## Verification

```text
GOTOOLCHAIN=local go test ./... -count=1
GOTOOLCHAIN=local go test -race ./... -count=1
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./...
```
