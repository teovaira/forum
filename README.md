# Forum

A web forum built in Go, backed by SQLite. Registered users can create posts tagged with one or more
categories, comment on posts, and like/dislike posts and comments; visitors can read everything but
must register to participate. Posts can be filtered by category, by the logged-in user's own posts, or
by posts the logged-in user has liked.

The category set is anime-themed, grouped into four kinds — demographic (Shonen, Shoujo, Seinen,
Josei), genre (Action, Romance, Isekai, Mecha, Slice of Life, Fantasy, Horror, Comedy, Sports), theme
(School, Music, Military, Supernatural, Historical), and discussion (Anime Discussion, Manga,
Recommendations, News, Fan Creations, General) — but the underlying mechanism is generic tagging and
filtering: any post can carry one or more categories from any group.

## Features

- Registration and login with bcrypt-hashed passwords and UUID cookie-based sessions (one active
  session per user, timestamps always UTC).
- Create posts with one or more categories; comment on posts.
- Like/dislike posts and comments, with live counts visible to everyone; liking a post you've already
  liked removes the reaction, disliking it replaces the like.
- Filter the post feed by category, by "my posts", or by "posts I've liked" (registered users only).
- Visitors (logged-out users) can read all posts and comments but cannot post, comment, or react.
- Progressive, framework-free JavaScript enhancements (logout confirmation, relative timestamps, a
  post-body character counter) — the site is fully functional with JavaScript disabled.

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
- `make` (optional — every command below has a plain Go/Docker equivalent)

## Getting Started

The module is already initialized and committed (`go.mod`, `go.sum`) — clone and fetch, don't
re-run `go mod init`:

```bash
git clone <repo-url> && cd forum
go mod download
```

### Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port the server listens on |
| `DB_PATH` | `./forum.db` | Path to the SQLite database file |
| `SECURE_COOKIES` | off | Set `true` only when serving over HTTPS; must stay off for local HTTP — a `Secure` cookie is silently dropped by the browser over plain HTTP |

An empty value for `PORT` or `DB_PATH` is treated the same as unset — both fall back to their default
rather than producing an invalid value.

## Commands

Every task below is a plain Go (or Docker) command on the left, and its `make` shortcut on the
right. Neither is required — pick whichever you prefer; the `make` targets exist to save typing and
keep flags consistent, not to hide anything. Run `make help` at any time for the full target list.

### Run

```bash
go run ./cmd/server
```
```bash
make run                 # same, reading PORT/DB_PATH from your shell
make run PORT=9090       # override any variable on the command line
make dev                 # same as run, with SECURE_COOKIES explicitly off
```

Then visit `http://localhost:8080` (or your chosen `PORT`).

### Build

```bash
go build -o server ./cmd/server
```
```bash
make build
```

### Test

```bash
go fmt ./... && go vet ./... && go test ./... -v -cover -race
```
```bash
make test-cover          # identical, one command
make check               # the CI-safe variant: gofmt -l (fails loudly instead of
                          # rewriting files) + go vet + go test -race, no -v/-cover noise
make test                # go test ./... only
make test-race           # go test ./... -race only
```

Run a single test the same way either path — there's no shortcut for this one, since it's inherently
a one-off:

```bash
go test ./internal/auth/... -run TestHashPassword -v
```

### Formatting & Linting

```bash
go fmt ./...              # rewrites files in place
gofmt -l .                # lists unformatted files without changing them
go vet ./...
```
```bash
make fmt                  # go fmt ./...
make fmt-check            # gofmt -l ., non-zero exit if anything is unformatted
make vet                  # go vet ./...
make lint                 # fmt-check + vet
```

### Coverage Report

```bash
go test ./... -coverprofile=coverage.txt
go tool cover -html=coverage.txt -o coverage.html
```
```bash
make cover-html
```

### Docker

```bash
docker build -t forum:latest .
docker run -p 8080:8080 forum:latest
```
```bash
make docker               # build + run together
make docker-build         # build only
make docker-run           # run only (replaces any existing container of the same name)
make docker-stop          # stop and remove the running container
make docker-clean         # stop the container and remove its image
```

### Cleanup

```bash
rm -f server coverage.txt coverage.html forum.db
go clean
```
```bash
make clean
```

### Dependency Maintenance

```bash
go mod tidy
go mod verify
```
```bash
make tidy
```

## Routes

| Method | Path | Auth required |
| :--- | :--- | :--- |
| `GET` | `/` | No |
| `GET` | `/posts/{id}` | No |
| `GET`, `POST` | `/register` | No |
| `GET`, `POST` | `/login` | No |
| `POST` | `/logout` | Yes |
| `GET` | `/posts/new` | Yes |
| `POST` | `/posts` | Yes |
| `POST` | `/posts/{id}/comments` | Yes |
| `POST` | `/posts/{id}/like`, `/posts/{id}/dislike` | Yes |
| `POST` | `/comments/{id}/like`, `/comments/{id}/dislike` | Yes |
| `GET` | `/static/*` | No |

Every route is registered with Go 1.22's method-aware mux patterns, so a request with the wrong
method gets an automatic `405` rather than falling into handler code.

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
- [`LICENSE`](LICENSE) — project license
