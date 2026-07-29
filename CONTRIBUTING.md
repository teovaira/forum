# Contributing

This document restates the team workflow from [`ROADMAP (2).md`](ROADMAP%20(2).md) §1 for anyone
contributing to this repository. Read the roadmap first — it holds the full architecture, database
schema, and frozen shared contracts (types, function signatures, routes) that all code must follow.

## Ownership Rule

Do not edit a file you do not own without asking and receiving explicit permission from the owner
first. See ROADMAP §3 for the file-to-owner map. Cross-package work (e.g. posts code needing reaction
counts) is bridged by calling the owning package's functions, never by editing their files directly.

## Branching

- Branch off `dev`, never off `main` directly.
- Name branches `<username>/<feature>` (e.g. `theo/auth-sessions`, `marios/posts-comments`).
- Rebase onto `dev` before every push.
- Merge a feature into `dev` only once it's tested; `dev` merges into `main` only at the milestones
  defined in ROADMAP §5.
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

Strict red-green-refactor, one function at a time:

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

## Error Handling

Raw Go or SQL errors must never leak to the client:

- Catch expected failures (`UNIQUE` violations, not-found, malformed input) directly.
- Surface them as friendly messages: form pages via the `ErrorMessage` view-data field, everything
  else via `webutil.RenderError` with the correct HTTP status (`400` bad input, `401`/`403` auth,
  `404` missing, `500` unexpected).
- Each owner handles the constraint errors thrown by their own queries. Log the real error
  server-side; show a clean message to the user.

## Resource Hygiene

Every `*sql.Rows` from a `Query` call gets `defer rows.Close()` immediately after the error check;
every prepared `*sql.Stmt` gets `defer stmt.Close()`. The single `*sql.DB` handle from
`database.Connect` is never closed mid-request — only at server shutdown, if at all.

## Code Cleanliness

- Run `gofmt` and `goimports` before every commit.
- Every exported Go identifier (types, functions, variables) gets a doc comment explaining *why* it
  exists.
- New shared names (types, function signatures, API routes, form fields) must be proposed, agreed
  upon, and added to ROADMAP §4 before anyone implements against them.

## Before Opening a Pull Request

- [ ] `make check` (or `go fmt ./... && go vet ./... && go test ./... -v -cover -race`) passes
- [ ] Test committed before implementation (backend)
- [ ] Commit messages follow `type(scope): description`
- [ ] Branch rebased onto latest `dev`
- [ ] Only files you own were touched
- [ ] Any new shared name was added to ROADMAP §4 before use
