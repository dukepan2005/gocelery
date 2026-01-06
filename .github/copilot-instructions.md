# AI Coding Instructions for gocelery

## Project Snapshot
- Go client/worker for Celery-compatible task queues; supports Redis and AMQP transports per [../README.md](../README.md).
- Brokers/backends must speak JSON + base64 Celery protocol v1; protocol v2 helpers exist but Celery stacks still need `CELERY_TASK_PROTOCOL=1` unless you explicitly go through the experimental v2 flow in [../gocelery_v2.go](../gocelery_v2.go) and [../message_v2.go](../message_v2.go).
- Core abstractions live in [../gocelery.go](../gocelery.go): `CeleryBroker`, `CeleryBackend`, `CeleryClient`, `AsyncResult`, and the optional `CeleryTask` interface for struct-based tasks.

## Architecture & Data Flow
- `CeleryClient.Delay/DelayKwargs` builds a pooled `TaskMessage`, encodes it, and pushes it through the broker; workers consume, execute, then persist results via the backend ([../gocelery.go](../gocelery.go), [../worker.go](../worker.go)).
- `CeleryWorker` spins `numWorkers` goroutines that tick every 100ms and invoke the broker’s non-blocking `GetTaskMessage`; keep new broker implementations non-blocking to preserve this design ([../worker.go](../worker.go)).
- Tasks execute either via reflection over plain functions or the `CeleryTask` interface (`ParseKwargs` + `RunTask`) for zero-reflection workflows ([../worker.go](../worker.go)).
- Message structs (`CeleryMessage`, `TaskMessage`, `ResultMessage`) are pooled via `sync.Pool`; always pair `get*` with the corresponding `release*` helper after use ([../message.go](../message.go), [../message_v2.go](../message_v2.go)).
- Redis transports route through [../redis_broker.go](../redis_broker.go) and [../redis_backend.go](../redis_backend.go); AMQP equivalents mirror the same contract in [../amqp_broker.go](../amqp_broker.go) and [../amqp_backend.go](../amqp_backend.go).

## Workflow & Tooling
- Use the provided phony targets: `make build` (`go install .`), `make lint` (`golangci-lint run`), and `make test` (`go test -timeout 30s -v -cover .`) from [../Makefile](../Makefile).
- Go module metadata resides in [../go.mod](../go.mod) / [../go.sum](../go.sum); prefer `go test ./...` for full coverage when editing multiple packages.
- Examples for smoke testing live under [../example/](../example/); Python scripts show Celery-side configuration, while [../example/goclient](../example/goclient) and [../example/goworker](../example/goworker) showcase Go usage.

## Integration Nuances
- Brokers assume JSON serialization and require Celery to disable pickle and force UTC per the checklist in [../README.md](../README.md).
- Redis broker supports per-task queue overrides via the optional `TaskQueue map[string]string`; honor this when routing protocol v2 messages ([../redis_broker.go](../redis_broker.go)).
- AMQP components share a single channel; concurrent use of channels is avoided, so don’t spawn separate goroutines per channel unless you recreate the connection ([../amqp_broker.go](../amqp_broker.go)).
- Result storage: Redis backend writes `celery-task-meta-<taskID>` keys with 24h TTL, while AMQP backend derives queue names by stripping dashes from task IDs ([../redis_backend.go](../redis_backend.go), [../amqp_backend.go](../amqp_backend.go)).

## Coding Conventions
- Keep new broker/backends lightweight and non-blocking; workers poll aggressively and expect quick returns (return `nil, err` when empty, not a blocking read).
- Normalize argument types before invoking user tasks; current helpers already coerce JSON float64 to ints/floats—extend `runTaskFunc` carefully if adding more coercions ([../worker.go](../worker.go)).
- Respect message pooling helpers and avoid retaining references beyond the current scope to prevent reusing mutated structs.
- When adding protocol v2 support, populate headers via `getCeleryMessageHeadersV2` to ensure `Origin`, `Argsrepr`, and IDs stay consistent ([../message_v2.go](../message_v2.go)).
- Prefer `StartWorkerWithContext` when embedding workers inside larger services so callers can cancel via parent contexts ([../worker.go](../worker.go)).

## Testing & Debugging Tips
- Unit tests cover brokers/backends/workers (`*_test.go` files in root); follow their patterns when adding new transports or task execution logic.
- To reproduce cross-language issues quickly, run the Python worker/client under [../example/](../example/) alongside the Go worker/client to verify serialization expectations.
- Async issues usually stem from broker queue mismatches—inspect Redis lists with `LRANGE celery 0 -1` or enable AMQP queue declaration logs to confirm routing keys.

Let me know if any section feels incomplete or unclear and I can refine it further.
