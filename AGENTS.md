# Change workflow

Make changes on a local topic branch so the user can review and try them before committing. A request to implement a change does not by itself authorize staging, committing, pushing, opening a pull request, or merging.

1. Start from an up-to-date `main` branch and create a topic branch. Never make or commit changes directly on `main`.
2. Use the branch prefixes enforced by CI: `feat/`, `fix/`, `ci/`, `docs/`, `chore/`, `refactor/`, `test/`, `perf/`, or `build/`.
3. Make the requested changes and perform checks appropriate to the request. Leave changes unstaged and uncommitted by default, and summarize what changed so the user can review and test locally.
4. Stage and commit only when the user explicitly asks for a commit. Use a focused commit and a conventional subject, such as `fix: limit event type history to one week`.
5. Push a branch only when the user explicitly asks to push. Open a pull request only when the user explicitly asks for one, targeting `main`. A pushed branch alone is not a pull request.
6. For a requested PR, confirm its title and every commit subject follow the CI convention: `<type>(optional scope): description`, using one of the allowed types above. Wait for required CI checks to finish and resolve failures before merging.
7. Merge only when the user explicitly asks, and merge through the pull request. Never fast-forward or push commits directly to `main`.
8. After an explicitly requested merge, switch to `main`, pull the merged changes, and delete the topic branch locally and remotely if appropriate.

Treat commit, push, PR creation, and merge as separate actions. Do not infer permission for any of them from a request to implement a change or from permission to perform an earlier step. If a requested step is blocked, leave the work at its current reviewable state and report the blocker.
