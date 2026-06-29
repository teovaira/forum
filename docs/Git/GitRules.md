# Rules

## Branching Rules

### Base Rules
- Always branch off `dev`.
- Always finish your feature branch with `main`.
- The `dev` branch is merged into `main` only at specified milestones (checkpoints).
- When creating a branch, name it like `<username>/<feature>`.


### Rules for Team Developers
- If there are multiple developers working on a feature, you must create a Pull Request to merge your branch into `dev`.
- In the Pull Request, you must:
  1. **Explain clearly what you have done**.
  2. **Show screenshots or videos** if the changes are related to the UI/UX.
  3. **Tag the people** who will review your code.
- **For small fixes and patches**, you can update your branch and then push directly to `dev` using the `--force-with-lease` flag (this is only allowed for small fixes).

### Rules for the `dev` Branch
- The `dev` branch should always be clean, stable, and ready for other developers to use.
- **Do not push untested code** to `dev`.
- **Do not push code that breaks the build** to `dev`.


#### Branching Strategy
We use a per-feature branching strategy branching off the `dev` branch. Branches must be named using the convention `username/feature` to explicitly namespace work based on Git tooling conventions.

*   **Examples:**
    *   `theo/auth-sessions`
    *   `marios/posts-comments`
    *   `vasiliki/reactions`
    *   `krysta/templates`
*   If a feature requires multiple phases, open sequential branches in turn.
*   **Requirement:** Always `rebase` onto `dev` before every push. Only merge a feature into `dev` after it has been fully tested. The `dev` branch is merged into `main` only at designated project checkpoints.

#### EXAMPLE:

```bash
# Branch per feature, off dev
git checkout -b theo/auth-sessions dev      

# ...commit work...

# Rebase onto dev before every push
git fetch origin && git rebase origin/dev   
git push origin theo/auth-sessions

# When the feature is ready and tested:
git checkout dev && git pull
git merge theo/auth-sessions
git push origin dev
```

### Branch Protection
- **Main branch is protected**
    - Only a few people can push to main
    - Protected by a required number of approvals in the Pull Request (4)

- **Dev branch is protected**
    - Protected by a required number of approvals in the Pull Request (2)

- **Feature branches are not protected**
    - With this in mind, you MUST respect the **Ownership Rule** and dont overwrite each other's branches without permission.

- **Commit Rules**
    - Use conventional commits format.
    - `<type>(<optional scope>): <description>`
    
    
    