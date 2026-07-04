# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project

A Go 1.22 web forum backed by SQLite (`mattn/go-sqlite3`, CGO), server-rendered with `html/template`
— no frontend framework. Full architecture, database schema, frozen function signatures, routes, and
the phased build plan live in [`ROADMAP (2).md`](ROADMAP%20(2).md); read it before writing code. Do
not deviate from its Section 4 ("Shared Contracts") without updating that section first.

## Setup

```bash
go mod init forum
go get github.com/mattn/go-sqlite3 golang.org/x/crypto/bcrypt github.com/google/uuid
```

Requires Go 1.22+ and a C compiler (`gcc`, for the SQLite CGO bindings).

## Build, Run, Test

```bash
go run ./cmd/server                                             # run the server locally
go fmt ./... && go vet ./... && go test ./... -v -cover -race   # full check before every commit
go test ./internal/auth/... -run TestHashPassword -v            # run a single test
docker build -t forum:latest . && docker run -p 8080:8080 forum:latest
```

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

- `cmd/server/main.go` — wires `database.Connect`/`Init` → `auth.WithUser` (global middleware) →
  `auth.RegisterRoutes` → `content.RegisterRoutes`.
- `internal/database/` — connection setup (`PRAGMA foreign_keys = ON` at connect time) and the
  append-only `schema.sql`.
- `internal/models/` — field-only structs, no behavior.
- `internal/auth/` — password hashing, sessions (one per user, enforced by a `UNIQUE` constraint),
  cookie middleware, `/register` `/login` `/logout`.
- `internal/content/` — posts, comments, categories, reactions. Posts/comments code never writes its
  own reaction or category SQL — it calls the read helpers `CountReactions`, `PostIDsInCategory`,
  `PostIDsLikedByUser` (frozen in ROADMAP §4.3) instead, to keep the two halves of this package
  conflict-free.
- `internal/webutil/` — `RenderTemplate`, `RenderError`; used by every handler in every other package.
- `web/templates/`, `web/static/` — HTML templates and CSS/JS assets (BEM class naming).

## Commit Convention

`<type>(<scope>): <description>` — types: `feat, fix, test, refactor, docs, style, chore, build`.
Scope is the package/area touched (`auth`, `content`, `database`, `templates`, `css`, `docker`). One
logical change per commit; see [`CONTRIBUTING.md`](CONTRIBUTING.md) for the full workflow.

## Before Submitting a Change

Run `go fmt ./... && go vet ./... && go test ./... -v -cover -race` and confirm it's clean. Check that
any new shared type, function signature, route, or form field name matches (or has been added to)
ROADMAP §4 before relying on it elsewhere.
