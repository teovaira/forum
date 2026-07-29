# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Project roadmap covering architecture, database schema, shared contracts, and phased build plan.
- Root-level documentation: `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `LICENSE`.
- SQLite schema (`users`, `sessions`, `posts`, `comments`, `categories`, `post_categories`,
  `reactions`), seeded with a fixed anime-themed category set (demographic/genre/theme/discussion).
- Authentication: registration and login with bcrypt-hashed passwords, UUID session tokens, one
  active session per user, cookie-based middleware (`WithUser`, `RequireAuth`), `/register` `/login`
  `/logout`.
- Content: creating posts with one or more categories, commenting on posts, filtering the post feed
  by category, by the logged-in user's own posts, and by posts they've liked.
- Reactions: liking/disliking posts and comments with a single toggling upsert (same value again
  removes the reaction; opposite value replaces it), batched reaction-count and category-lookup
  helpers to avoid N+1 queries.
- Shared rendering layer (`internal/webutil`): buffer-first template rendering so a failed render
  never sends a half-written page, and a plain-text fallback if the error page itself fails to render.
- Server-rendered HTML templates for the home feed, a single post with its comments, login,
  registration, the new-post form, and a generic error page; component CSS on a design-token system.
- Progressive, framework-free JavaScript (`web/static/js/main.js`): a logout confirmation step,
  relative timestamps ("3 minutes ago"), and a live character counter on the new-post form — every
  feature degrades to the plain server-rendered page with JavaScript disabled.
- `cmd/server/main.go`: wires configuration, the database, the template set, and every package's
  routes into a runnable server.
- End-to-end HTTP tests (`cmd/server`) driving the real router over a live `httptest.NewServer`,
  covering the full register → post → comment → react → logout journey and the audit's direct-SQL
  verification checks.
- `Makefile` with build/run/test/lint/docker targets (`make help` lists all of them), satisfying the
  "script to build the images and containers" requirement once paired with the Dockerfile.

### Fixed
- Session timestamps are now stored and compared in UTC everywhere. `expires_at` is compared as a
  plain SQL string, and a local-time offset does not sort correctly against a UTC "Z" suffix — a
  valid session could read as already expired on any server not already running in UTC.
- Foreign key enforcement is set via a connection DSN flag instead of a one-time `PRAGMA` after
  opening, so it applies to every connection a pool opens, not just the first.

### Known gaps
- No `Dockerfile` yet — the spec's Docker requirement and the audit's container-build checks are not
  yet satisfiable.

[Unreleased]: https://platform.zone01.gr/git/mvidenma/Forum/compare/main...HEAD
