# cacheDB

A from-scratch, in-progress clone of Redis written in Go. It runs a synchronous TCP
server that speaks (a subset of) the RESP (REdis Serialization Protocol), backed by
an in-memory keyspace with TTL/expiry, on-demand snapshot persistence, and a basic
key-count eviction policy.

This is an early-stage learning project, built iteratively one command/feature at a
time. It's compatible enough with RESP to be driven directly by `redis-cli`.

## What it does

- **In-memory key-value store** — `SET`, `GET`, `DEL`, `EXISTS`.
- **TTL / expiry** — `EXPIRE`, `TTL`, plus `SET ... EX seconds` / `SET ... PX milliseconds`.
  Expiry is passive: a key is checked and removed the next time it's accessed, not on a
  background timer.
- **Persistence** — `SAVE` writes a snapshot of the keyspace to disk (`dump.cachedb`),
  which is automatically reloaded on server startup. There is no periodic/background
  snapshotting and no append-only-file (AOF) log.
- **Eviction** — an optional key-count cap (`-maxkeys`). When exceeded, the key with the
  soonest expiry is evicted first; if no key has a TTL, an arbitrary key is evicted.
  This is not real LRU/LFU.
- **Connection model** — one goroutine per TCP connection, fully synchronous/blocking;
  no event loop, no worker pool.

## Architecture

The code is a small pipeline:

```
TCP connection → RESP decode → command parse → eval/dispatch (+ keyspace) → RESP encode → response write
```

| Package / file            | Responsibility |
|----------------------------|----------------|
| `main.go`                  | Entrypoint. Parses `-host`/`-port`/`-maxkeys` flags, loads any on-disk snapshot via `core.Load()`, starts the server. |
| `config/config.go`         | Package-level mutable config (`Host`, `Port`, `MaxKeys`) shared across the app instead of a passed-around struct. |
| `server/sync_tcp.go`       | `RunSyncTCPServer` accepts connections and spawns a goroutine per connection. Each connection buffers incoming bytes and repeatedly calls `core.DecodeOne` to peel off complete RESP messages, since a message can arrive split across multiple TCP reads. |
| `core/resp.go`             | The RESP codec. `DecodeOne` dispatches on the leading byte (`+ - : $ *`) to per-type readers, each returning `(value, bytesConsumed, error)`. `Encode` type-switches Go values back into RESP wire format. |
| `core/ParseCommand.go`     | Converts a decoded RESP array into a `*RedisCmd`, uppercasing the command name. |
| `core/type.go`             | `RedisCmd{Cmd string, Args []string}` — the internal command representation. |
| `core/store.go`            | The keyspace: a `sync.RWMutex`-guarded `map[string]entry`, where `entry{value, expiresAt}` (zero `expiresAt` = no TTL). Accessors: `setKey`, `setKeyWithTTL`, `getKey` (passively expires on read), `deleteKey`, `expireKey`, `ttlSeconds`, and `evictIfNeeded` (runs inside the lock at the end of every `setKey*`, gated by `config.MaxKeys > 0`). |
| `core/persistence.go`      | `Save()` / `Load()` gob-encode/decode the keyspace to/from `dump.cachedb`. Expired keys are excluded from snapshots; loading a missing file is a no-op. |
| `core/eval.go`             | `EvalAndRespond` switches on `cmd.Cmd` and dispatches to `eval<CMD>` functions, each of which writes an encoded RESP response directly to the `net.Conn`. |

### Supported commands

`PING`, `ECHO`, `SET` (optional `EX seconds` / `PX milliseconds`), `GET`, `DEL`,
`EXISTS`, `EXPIRE`, `TTL`, `SAVE`.

## Running it

```bash
# Run the server (defaults to 0.0.0.0:8080, unlimited keys)
go run main.go

# Run with a custom host/port and an eviction cap
go run main.go -host 127.0.0.1 -port 6380 -maxkeys 1000

# Build a standalone binary
go build -o cachedb .
```

Connect with `redis-cli` (or `nc`), since the server speaks RESP:

```bash
redis-cli -p 8080
> SET foo bar
OK
> GET foo
"bar"
> EXPIRE foo 100
(integer) 1
> TTL foo
(integer) 100
> SAVE
OK
```

## How to check / test it

```bash
# Compile-check everything
go vet ./...

# Format check
gofmt -l .

# Run the full test suite (core package)
go test ./...

# Run a single test
go test ./core -run TestEvalEXPIRE -v
```

Tests are split by concern in the `core` package:

- `store_test.go`, `eviction_test.go`, `persistence_test.go` exercise the keyspace,
  eviction, and snapshot helpers directly (no network involved).
- `eval_test.go` exercises command handlers end-to-end over `net.Pipe()`, simulating a
  real client connection without opening a socket.

No CI config exists yet — `go vet ./...` and `go test ./...` are the manual gate before
committing.
