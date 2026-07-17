# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **Always read [@specs.txt](specs.txt), [@audit.txt](audit.txt), and [@ROADMAP (2).md](ROADMAP%20(2).md)
> before doing any work in this repository.** `specs.txt` is the assignment brief (what must be built),
> `audit.txt` is the grading checklist the finished project must satisfy, and `ROADMAP (2).md` is the
> frozen architecture, database schema, shared contracts, and phased build plan. They are the source of
> truth; this file only summarizes them.

## Project Status

This is a Go web forum (the classic 01-edu/Zone01 "forum" project, extended with an anime-community
theme). **No application code exists yet** — `cmd/`, `internal/`, and `web/` currently contain only
`.gitkeep` placeholders. The full design is frozen in [`ROADMAP (2).md`](ROADMAP%20(2).md) at the repo
root: read it before writing any code. It is the source of truth for schema, model fields, function
signatures, routes, and phase order — treat Section 4 ("Shared Contracts") as an API you must not
deviate from without updating that section first. `specs.txt` is the original assignment brief;
`audit.txt` is the grading checklist the finished project must satisfy.

## Commands

```bash
go mod init forum                                                    # already done
go get github.com/mattn/go-sqlite3 golang.org/x/crypto/bcrypt github.com/google/uuid

go run ./cmd/server                                                  # run local server
go fmt ./... && go vet ./... && go test ./... -v -cover -race        # test & sanitize (run before every commit)
go test ./internal/auth/... -run TestHashPassword -v                 # single test example

docker build -t forum:latest .                                       # containerize
docker run -p 8080:8080 forum:latest                                 # run container
```

Env vars: `PORT` (default `8080`), `DB_PATH` (default `./forum.db`), `SECURE_COOKIES` (default off;
set `true` only when serving HTTPS).

Prerequisites: Go 1.22+, a C compiler (`gcc`, required for `mattn/go-sqlite3` CGO bindings), Docker.

## Architecture

**Module:** `forum` (Go 1.22).

**Package layout** (fixed — see ROADMAP §3 for full file-level ownership map):
- `cmd/server/main.go` — wires everything: `database.Connect`/`Init` → `auth.WithUser` middleware
  applied globally → `auth.RegisterRoutes` → `content.RegisterRoutes`.
- `internal/database/` — `database.go` (`Connect(path)`, `Init(db)` which runs `schema.sql`) and the
  append-only `schema.sql`. `PRAGMA foreign_keys = ON` is set at connection time in `database.go`, not
  in the schema file itself.
- `internal/models/` — plain field-only structs, no behavior (`User`, `Session`, `Category`, `Post`,
  `Comment`, `ReactionValue`, `ReactionTarget`). Exact fields are frozen in ROADMAP §4.2.
- `internal/auth/` — password hashing, sessions, cookie-based middleware, `/register` `/login`
  `/logout` handlers.
- `internal/content/` — posts, comments, categories, and reactions. Two people's work shares this one
  package (posts/comments vs. categories/reactions) — see the cross-package interface below.
- `internal/webutil/` — `RenderTemplate`, `RenderError`; the only place that talks to `html/template`
  and writes error responses. Used by every handler in every other package.
- `web/templates/` — `layout.html`, `home.html`, `post.html`, `login.html`, `register.html`,
  `new_post.html`, `error.html`.
- `web/static/css/` — `tokens.css` (design tokens/custom properties) loaded before `style.css`
  (component rules built on those tokens). BEM class naming (`post__title`,
  `post__reaction-button--active`).

**Database schema** (`internal/database/schema.sql`, ROADMAP §4.1): `users`, `sessions` (one row per
`user_id`, enforced via `UNIQUE` — logging in again replaces the old session, never stacks a second
one), `categories` (fixed seeded list with a `kind` CHECK constraint grouping tags into
demographic/genre/theme/discussion — see ROADMAP §7 for the full rationale), `posts`,
`post_categories` (many-to-many junction; every post needs ≥1 category, `General` is the safe
default), `comments`, and a single unified `reactions` table covering both posts and comments via a
`target_type` CHECK column instead of two near-identical tables (`UNIQUE(user_id, target_id,
target_type)` enforces one reaction per user per target). Toggle semantics for `UpsertReaction`: same
value again → delete row (undo); opposite value → update in place; no existing row → insert.

**The one cross-package interface that matters:** posts/comments code (`ListPosts`, `GetPost`,
`filters.go`) never writes its own reaction or category SQL. It calls read helpers from the
categories/reactions side of `internal/content`: `CountReactions(db, target, ids)` (batched, avoids
N+1), `PostIDsInCategory(db, categoryID)`, `PostIDsLikedByUser(db, userID)`. These signatures are
frozen in ROADMAP §4.3 specifically so both halves can be implemented in parallel against an agreed
contract.

**Routing:** Go 1.22 method-aware `mux.HandleFunc("GET /path", ...)` patterns are required (not the
old method-less form) so `http.ServeMux` returns `405` automatically on a method mismatch — don't rely
on manually branching on `r.Method` inside a handler as the only guard. Full route table is ROADMAP
§4.5.

**Auth/session model:** `auth.WithUser` is applied globally and populates the request context if a
valid `session_token` cookie exists; `auth.RequireAuth` wraps only protected routes and blocks with a
clean `401`/redirect (never a bare status) if no user is in context. The user is read back out via
`auth.UserFromContext(ctx)` — the context key itself is private to the `auth` package. Cookie
attributes: `HttpOnly; SameSite=Strict; Path=/` always on; `Secure` is conditional on `SECURE_COOKIES`
and must stay off for local `http://localhost` testing (a `Secure` cookie is silently dropped over
plain HTTP, which would break login during grading).

**Error handling:** raw Go/SQL errors must never reach the client. Form pages (auth, new post/comment)
surface failures via an `ErrorMessage` field on the page's view-data struct (ROADMAP §4.4); everything
else calls `webutil.RenderError(w, statusCode, message)` with `400` (bad input), `401`/`403` (auth),
`404` (missing), or `500` (unexpected). Log the real error server-side; show a clean string to the
user. Every required form field (registration, login, post title/body, comment body) must be rejected
server-side when empty/whitespace-only — don't rely on client-side validation alone.

**Resource hygiene (audited explicitly):** every `*sql.Rows` gets `defer rows.Close()` immediately
after the error check; every `*sql.Stmt` gets `defer stmt.Close()`; the single `*sql.DB` handle from
`database.Connect` is only ever closed at server shutdown, never mid-request.

## Team Conventions

Full detail lives in [`CONTRIBUTING.md`](CONTRIBUTING.md) (workflow, branching, commits) and
[`AGENTS.md`](AGENTS.md) (agent-facing conventions), with the complete rationale in ROADMAP §1. Key
points that affect how you should write code in this repo:

- **Ownership Rule:** don't edit a file you don't own without asking the owner first (see ROADMAP §3
  for the file→owner map). The Marios↔Vasiliki content-package split is bridged by calling the other
  person's functions, never by editing their files.
- **TDD for backend Go code:** write a failing test → commit → minimum implementation to pass → commit.
  Never combine test and implementation in one commit; never write implementation before the test
  exists. Frontend (HTML/CSS) skips red/green — build, verify in browser, commit.
- **Commit format:** `<type>(<scope>): <description>` — types are `feat, fix, test, refactor, docs,
  style, chore, build`; scope is the package/area touched (`auth`, `content`, `database`, `templates`,
  `css`, `docker`).
- **Every exported Go identifier needs a doc comment explaining *why* it exists**, not what it does.
- **New shared names** (types, function signatures, routes, form field names) must be added to ROADMAP
  §4 before anyone writes code against them — propose, agree, document, then implement.
- **Branching:** per-feature branches off `dev`, named `username/feature`; rebase onto `dev` before
  every push; `dev` merges to `main` only at the phase checkpoints in ROADMAP §5. Never commit directly
  to `dev` or `main`.
