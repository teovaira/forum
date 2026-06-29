# Workflow & Git Strategy

## Test-Driven Development (TDD) for Go Backend
We follow a strict red-green-refactor discipline for backend code:

1.  **Red:** Write a failing test $\rightarrow$ **Commit**.
2.  **Green:** Write the minimum implementation code to pass the test $\rightarrow$ **Commit**.

> [!WARNING]
> **Rule:** Never combine the test and implementation in a single commit. Never write production code before the test exists.

## Commit Message Format
We use a structured conventional commit format: `<type>(<scope>): <description>`

| Commit Type | Description |
| :--- | :--- |
| `feat` | A new feature |
| `fix` | A bug fix |
| `test` | Adding or correcting tests |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `docs` | Documentation only changes |
| `style` | Changes that do not affect the meaning of the code (white-space, formatting, etc.) |
| `chore` | Updating build tasks, package manager configs, etc. |
| `build` | Changes that affect the build system or external dependencies |

*   **Scopes:** The specific package or area touched (e.g., `auth`, `content`, `database`, `templates`, `css`, `docker`).

## Frontend Workflow (HTML/CSS)
Follow a rapid, small-steps discipline:
1.  Build the layout/style.
2.  Verify layout and functionality directly in the browser.
3.  **Commit** using `feat` or `style` types.

## Error Handling and User Experience
Raw Go or SQL system errors must never leak to the client application.

*   Catch expected database/system failures directly (such as `UNIQUE` constraint violations, item not found, or malformed input).
*   Surface them as user-friendly messages: Form pages handle errors via the `ErrorMessage` view field; all other layers use `webutil.RenderError` with the appropriate HTTP status code (`400` Bad Request, `401`/`403` Auth, `404` Missing, `500` Server Error).
*   **Ownership:** Each code owner is responsible for catching and handling constraint errors thrown by their own queries. Log the actual internal error server-side for personal debugging; show a clean, descriptive string to the end user.

## Architectural Decision Log
We maintain a lightweight architecture record named `ai_changelog.md` at the repository root to capture technical reasoning.

> [!IMPORTANT]
> **Rule:** Each architectural decision entry must be bundled **in the exact same commit** as the code implementing it. Never create standalone changelog-only commits.

**Format:**
```markdown
## YYYY-MM-DD — <name>
Decision: <What was decided>
Reason: <Why this approach was taken>
```

## Code Cleanliness and Documentation
*   Run `gofmt` and `goimports` to clean up source files before every single commit.
*   Every exported Go identifier (types, functions, variables) must include a doc comment explaining *why* it exists.
*   New shared names (types, function signatures, API routes, form fields) must be explicitly proposed, agreed upon, and documented prior to implementation.
