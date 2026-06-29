# Reorganize Documentation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use advancedSkills:executing-plans to implement this plan task-by-task.

**Goal:** Split the monolithic CODE_OF_CONDUCT.md file into logical markdown files under docs/ and convert CODE_OF_CONDUCT.md into a landing page with relative markdown file links.

**Architecture:** Create separate subdirectories under docs/ (Standards/, Workflow/, Setup/, Team/), move content into corresponding files, and transform CODE_OF_CONDUCT.md to serve as an index/navigation menu linking to these docs.

**Tech Stack:** Markdown

---

### Task 1: Create docs/Standards/Pledge.md

**Files:**
- Create: `docs/Standards/Pledge.md`

**Step 1: Write the content**
Write the following content to `docs/Standards/Pledge.md`:
```markdown
# Our Pledge

We are committed to providing a collaborative, respectful, and high-quality development environment for our team. To maintain order, prevent merge conflicts, and respect individual contributions, we adhere strictly to the following fundamental rule:

> [!IMPORTANT]
> **Ownership Rule:** Do not edit a file you do not own without asking and receiving explicit permission from the owner first.
```

**Step 2: Verify the file exists and is populated**
Check the file content to ensure it is written properly.

**Step 3: Commit**
```bash
git add docs/Standards/Pledge.md
git commit -m "docs: extract Pledge and Ownership Rule section"
```

---

### Task 2: Create docs/Workflow/Workflow.md

**Files:**
- Create: `docs/Workflow/Workflow.md`

**Step 1: Write the content**
Write the following content to `docs/Workflow/Workflow.md`:
```markdown
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
```

**Step 2: Verify the file exists and is populated**
Check the file content.

**Step 3: Commit**
```bash
git add docs/Workflow/Workflow.md
git commit -m "docs: extract Workflow and Standards section"
```

---

### Task 3: Create docs/Setup/Setup.md

**Files:**
- Create: `docs/Setup/Setup.md`

**Step 1: Write the content**
Write the following content to `docs/Setup/Setup.md`:
```markdown
# Setup & Installation

## Prerequisites
*   **Go** 1.22+
*   **C Compiler** (`gcc` — required for `mattn/go-sqlite3` CGO bindings)
*   **Git**
*   **Docker**

## Initializing the Project
```bash
git clone <repo-url> && cd forum
go mod init forum
go get github.com/mattn/go-sqlite3 golang.org/x/crypto/bcrypt github.com/google/uuid
```

## Environment Variables
*   `PORT`: Sets the server network port (Default: `8080`)
*   `DB_PATH`: Sets the local path to the SQLite database file (Default: `./forum.db`)

## Core Commands

| Action | Command |
| :--- | :--- |
| **Run Local Server** | `go run ./cmd/server` |
| **Test & Sanitize** | `go fmt ./... && go vet ./... && go test ./... -v -cover -race` |
| **Containerize App** | `docker build -t forum:latest .` |
| **Run Container** | `docker run -p 8080:8080 forum:latest` |
```

**Step 2: Verify the file exists and is populated**
Check the file content.

**Step 3: Commit**
```bash
git add docs/Setup/Setup.md
git commit -m "docs: extract Setup & Installation section"
```

---

### Task 4: Create docs/Team/Team.md

**Files:**
- Create: `docs/Team/Team.md`

**Step 1: Write the content**
Write the following content to `docs/Team/Team.md`:
```markdown
# Team Responsibilities

Workload is balanced equally across the team ($\approx 25\%$ per member). Functional roles represent ownership domains rather than rigid silos—every member is expected to contribute across stack boundaries as needed:

| Owner | Primary Domain | Description / Responsibilities |
| :--- | :--- | :--- |
| **Theo** | Backend Core & Security | Database layer setup, user authentication, and session handling. |
| **Marios** | Application Logic & Ops | Post/comment mechanics, application containerization (Docker). |
| **Vasiliki** | Schema & Design System | Database schema design, SQL queries for categories/reactions, frontend design tokens. |
| **Krysta** | Presentation & Styling | Render pipeline, templates execution, component CSS architecture, backend scaffolding. |
```

**Step 2: Verify the file exists and is populated**
Check the file content.

**Step 3: Commit**
```bash
git add docs/Team/Team.md
git commit -m "docs: extract Team Responsibilities section"
```

---

### Task 5: Modify CODE_OF_CONDUCT.md to serve as an index

**Files:**
- Modify: `CODE_OF_CONDUCT.md`

**Step 1: Overwrite the content**
Overwrite `CODE_OF_CONDUCT.md` with:
```markdown
# Project Documentation & Development Standards

Welcome to the project's developer guidelines, workflows, and documentation. Below you can find links to individual sections covering standards, setup instructions, responsibilities, and workflows.

## Standards & Code of Conduct
- [Our Pledge](file:///var/home/marios/Documents/Cohort/Forum/docs/Standards/Pledge.md) - Our values and the Ownership Rule.
- [Team Workflow & Standards](file:///var/home/marios/Documents/Cohort/Forum/docs/Workflow/Workflow.md) - Code quality, branching strategy, TDD, conventional commits, error handling, and architectural decisions.
- [Setup & Installation](file:///var/home/marios/Documents/Cohort/Forum/docs/Setup/Setup.md) - Environment details, project initialization, and core commands.
- [Team Responsibilities](file:///var/home/marios/Documents/Cohort/Forum/docs/Team/Team.md) - Workload balancing and domain ownership breakdown.

## Other Documentation
- [Git branching rules](file:///var/home/marios/Documents/Cohort/Forum/docs/Git/GitRules.md) - Guidelines for branching and merging.
- [AI logs](file:///var/home/marios/Documents/Cohort/Forum/docs/ailogs/aiLog.md) - AI workflow and prompt details.
- [README](file:///var/home/marios/Documents/Cohort/Forum/README.md) - Main repository readme.
```

**Step 2: Verify links**
Check that the links match the expected paths.

**Step 3: Commit**
```bash
git add CODE_OF_CONDUCT.md
git commit -m "docs: turn CODE_OF_CONDUCT.md into navigation index"
```
