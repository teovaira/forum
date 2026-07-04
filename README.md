# Forum

A web forum built in Go, backed by SQLite. Registered users can create posts tagged with one or more
categories, comment on posts, and like/dislike posts and comments; visitors can read everything but
must register to participate. Posts can be filtered by category, by the logged-in user's own posts, or
by posts the logged-in user has liked.

The category set is anime-themed (demographic/genre/theme/discussion tags — see `ROADMAP (2).md` §7
for the rationale), but the underlying mechanism is generic tagging and filtering.

## Features

- Registration and login with hashed passwords and cookie-based sessions (one active session per user).
- Create posts with one or more categories; comment on posts.
- Like/dislike posts and comments, with live counts visible to everyone.
- Filter the post feed by category, by "my posts", or by "posts I've liked".
- Visitors (logged-out users) can read all posts and comments but cannot post, comment, or react.

## Tech Stack

- Go 1.22+ (standard library `net/http`, method-aware routing)
- SQLite via `mattn/go-sqlite3` (CGO)
- `golang.org/x/crypto/bcrypt` for password hashing
- `google/uuid` for session tokens
- Server-rendered HTML (`html/template`), no frontend framework
- Docker for containerized builds/runs

## Prerequisites

- Go 1.22+
- A C compiler (`gcc`) — required for the `mattn/go-sqlite3` CGO bindings
- Docker (for containerized runs)

## Getting Started

```bash
git clone <repo-url> && cd forum
go mod init forum
go get github.com/mattn/go-sqlite3 golang.org/x/crypto/bcrypt github.com/google/uuid
```

### Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port the server listens on |
| `DB_PATH` | `./forum.db` | Path to the SQLite database file |
| `SECURE_COOKIES` | off | Set `true` only when serving over HTTPS; must stay off for local HTTP |

### Run

```bash
go run ./cmd/server
```

Then visit `http://localhost:8080`.

### Test

```bash
go fmt ./... && go vet ./... && go test ./... -v -cover -race
```

### Docker

```bash
docker build -t forum:latest .
docker run -p 8080:8080 forum:latest
```

## Project Structure

```
cmd/server/        entry point, wires database/auth/content together
internal/auth/     password hashing, sessions, cookie middleware, auth routes
internal/content/  posts, comments, categories, reactions
internal/database/ SQLite connection setup and schema
internal/models/   shared data types (User, Post, Comment, Category, ...)
internal/webutil/  shared template rendering and error responses
web/templates/     HTML templates
web/static/        CSS and JS assets
```

## Documentation

- [`AGENTS.md`](AGENTS.md) — guidance for AI coding assistants working in this repo
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — workflow, branching, and commit conventions
- [`CHANGELOG.md`](CHANGELOG.md) — release notes
- [`ROADMAP (2).md`](ROADMAP%20(2).md) — full architecture, schema, and phased implementation plan
- [`LICENSE`](LICENSE) — project license
