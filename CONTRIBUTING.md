# Contributing

This document describes the team workflow for anyone contributing to this repository.

## Ownership Rule

Do not edit a file you do not own without asking and receiving explicit permission from the owner
first. Cross-package work (e.g. posts code needing reaction counts) is bridged by calling the owning
package's functions, never by editing their files directly.

| Package | Owner |
| :--- | :--- |
| `internal/auth/`, `internal/database/` | Theo |
| `internal/content/` (posts, comments) | Marios |
| `internal/content/` (categories, reactions) | Vasiliki |
| `internal/webutil/`, `web/templates/`, `web/static/` | Krysta |
| `Dockerfile` | Marios |

## Branching

- Branch off `dev`, never off `main` directly.
- Name branches `<username>/<feature>` (e.g. `theo/auth-sessions`, `marios/posts-comments`).
- Rebase onto `dev` before every push.
- Merge a feature into `dev` only once it's tested; `dev` merges into `main` only at agreed
  milestones.
- Nobody commits directly to `dev` or `main`.

```bash
git checkout -b theo/auth-sessions dev
# ...commit work...
git fetch origin && git rebase origin/dev
git push origin theo/auth-sessions
# once tested:
git checkout dev && git pull
git merge theo/auth-sessions
git push origin dev
```

Small fixes/patches may be pushed directly to `dev` with `--force-with-lease` after rebasing; larger
changes go through a Pull Request that explains what was done, includes screenshots/video for UI
changes, and tags reviewers.

### Branch Protection

- `main` requires 4 approvals.
- `dev` requires 2 approvals.
- Feature branches are unprotected — respect the Ownership Rule instead of relying on protection.

## Test-Driven Development (backend Go code)

Strict red-green-refactor, one function at a time, as two separate commits:

1. **Red:** write a failing test → commit.
2. **Green:** write the minimum implementation to pass → commit.

Never combine a test and its implementation in a single commit. Never write production code before
the test exists.

Frontend (HTML/CSS) skips the red/green cycle — there's nothing to unit-test. Instead: build the
layout/style, verify it in the browser, then commit (`feat` or `style`).

## Commit Messages

Conventional commits: `<type>(<scope>): <description>`

| Type | Meaning |
| :--- | :--- |
| `feat` | A new feature |
| `fix` | A bug fix |
| `test` | Adding or correcting tests |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `docs` | Documentation only changes |
| `style` | Changes that don't affect code meaning (whitespace, formatting) |
| `chore` | Build tasks, package manager configs, etc. |
| `build` | Changes to the build system or external dependencies |

Scope is the package/area touched (`auth`, `content`, `database`, `templates`, `css`, `docker`).

## Code Conventions

Error handling, resource hygiene, doc comments, and formatting rules are documented once in
[`AGENTS.md`](AGENTS.md) — that file applies to every contributor, human or AI. Each package owner
handles the constraint errors thrown by their own queries.

## Before Opening a Pull Request

- [ ] `make check` (or `go fmt ./... && go vet ./... && go test ./... -v -cover -race`) passes
- [ ] Test committed before implementation (backend)
- [ ] Commit messages follow `type(scope): description`
- [ ] Branch rebased onto latest `dev`
- [ ] Only files you own were touched
- [ ] Any new shared name was agreed on by the team before use
