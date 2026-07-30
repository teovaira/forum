# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project

A Go 1.22 web forum backed by SQLite (`mattn/go-sqlite3`, CGO), server-rendered with `html/template`
— no frontend framework. Package layout, shared contracts (types, function signatures, routes), and
conventions are described below; don't deviate from a frozen signature without agreeing the change
with the team first.

## Setup

The module is already initialized (`go.mod` is committed) — clone and fetch, don't re-run `go mod init`:

```bash
git clone <repo-url> && cd forum
go mod download
```

Requires Go 1.22+ and a C compiler (`gcc`, for the SQLite CGO bindings).

## Build, Run, Test

Plain Go commands, with the equivalent `make` target alongside each:

```bash
go run ./cmd/server                                             # make run
go fmt ./... && go vet ./... && go test ./... -v -cover -race   # make test-cover
go test ./internal/auth/... -run TestHashPassword -v            # (no shortcut — one-off)
docker build -t forum:latest . && docker run -p 8080:8080 forum:latest   # make docker
```

`make check` is the CI-safe variant of the same gate: `gofmt -l` (fails loudly on unformatted files
instead of silently rewriting them) + `go vet` + `go test -race`, without `-v`/`-cover` noise. Run
`make help` for the full target list (build, run, test variants, lint, docker lifecycle) — see
`README.md` for the complete command reference.

Env vars: `PORT` (default `8080`), `DB_PATH` (default `./forum.db`), `SECURE_COOKIES` (default off;
must stay off for local HTTP — see cookie notes below).

## Code Conventions

- **TDD for backend Go code:** write a failing test, commit it, then write the minimum implementation
  to pass, commit that separately. Never combine test and implementation in one commit; never write
  implementation before the test exists.
- **Every exported identifier needs a doc comment explaining *why* it exists**, not a restatement of
  its name.
- Run `gofmt`/`goimports` before every commit.
- **Resource hygiene (checked in review/audit):** every `*sql.Rows` gets `defer rows.Close()`
  immediately after the error check; every `*sql.Stmt` gets `defer stmt.Close()`; the single `*sql.DB`
  handle from `database.Connect` is only closed at server shutdown, never mid-request.
- **Error handling:** never leak raw Go/SQL errors to the client. Form pages surface failures via an
  `ErrorMessage` field on the page's view-data struct; everything else calls
  `webutil.RenderError(w, statusCode, message)` with the correct HTTP status (400 bad input, 401/403
  auth, 404 missing, 500 unexpected). Log the real error server-side; show a clean message to the user.
- **Required fields** (registration, login, post title/body, comment body) must be rejected
  server-side when empty or whitespace-only — do not rely on client-side validation alone.
- **Routing:** use Go 1.22's method-aware mux patterns (`mux.HandleFunc("GET /path", ...)`), not the
  old method-less form, so mismatched methods get an automatic `405` instead of falling into handler
  code.
- **Cookies:** session cookie is `session_token` with `HttpOnly; SameSite=Strict; Path=/` always on;
  `Secure` is conditional on `SECURE_COOKIES` and must stay off for local `http://localhost` testing —
  a `Secure` cookie is silently dropped over plain HTTP.

## Architecture Map

- `cmd/server/main.go` — the entry point. Reads `PORT`/`DB_PATH`, calls `database.Connect`/`Init`,
  parses `web/templates/*.html` into `webutil.SetTemplates`, builds the router (`auth.RegisterRoutes`,
  `content.RegisterRoutes`, `webutil.RegisterRoutes` for `/static/*`), wraps it in `auth.WithUser`
  (applied globally — `content.RegisterRoutes` already applies `auth.RequireAuth` per-route, so
  `WithUser` must not be paired with a second global `RequireAuth`), then `http.ListenAndServe`.
- `internal/database/` — connection setup (foreign keys enabled via a DSN flag, so enforcement holds
  across every pooled connection, not just the first) and the append-only `schema.sql`.
- `internal/models/` — field-only structs, no behavior.
- `internal/auth/` — password hashing, sessions (one per user, enforced by a `UNIQUE` constraint;
  timestamps always UTC — see the note below), cookie middleware, `/register` `/login` `/logout`.
- `internal/content/` — posts, comments, categories, reactions. Posts/comments code never writes its
  own reaction or category SQL — it calls the read helpers `CountReactions`, `PostIDsInCategory`,
  `PostIDsLikedByUser` instead, to keep the two halves of this package conflict-free.
- `internal/webutil/` — `RenderTemplate`, `RenderError`; used by every handler in every other package.
- `web/templates/`, `web/static/` — HTML templates, CSS, and a small vanilla-JS file
  (`web/static/js/main.js`: logout confirmation, relative timestamps, a post-body character counter —
  no framework, and every feature degrades to the plain server-rendered behavior with JS disabled).

**Timestamps are always UTC.** Every `time.Now()` call that gets persisted (`created_at`,
`expires_at`, etc.) calls `.UTC()` before formatting. `expires_at` is compared as a plain SQL string
(`WHERE expires_at > ?`), and RFC 3339's timezone suffix does not sort correctly across offsets — a
mixed-timezone codebase would misjudge session expiry on any server not already running in UTC. Keep
this convention when adding a new timestamped column.

## Commit Convention

`<type>(<scope>): <description>` — types: `feat, fix, test, refactor, docs, style, chore, build`.
Scope is the package/area touched (`auth`, `content`, `database`, `templates`, `css`, `docker`). One
logical change per commit; see [`CONTRIBUTING.md`](CONTRIBUTING.md) for the full workflow.

## Before Submitting a Change

Run `go fmt ./... && go vet ./... && go test ./... -v -cover -race` and confirm it's clean. Check that
any new shared type, function signature, route, or form field name has been agreed on with the team
before relying on it elsewhere.
