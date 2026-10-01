# Change workflow

Use a pull request for every change to this repository, including code, documentation, and configuration changes.

1. Start from an up-to-date `main` branch and create a topic branch. Never make or commit changes directly on `main`.
2. Use the branch prefixes enforced by CI: `feat/`, `fix/`, `ci/`, `docs/`, `chore/`, `refactor/`, `test/`, `perf/`, or `build/`.
3. Make the requested changes and run relevant checks when appropriate. Keep commits focused and use a conventional subject, such as `ci: fix PR convention checks in workflow`.
4. Push the topic branch and open a pull request targeting `main`. A pushed branch alone is not a pull request.
5. Confirm the PR title and every commit subject follow the CI convention: `<type>(optional scope): description`, using one of the allowed types above. Wait for required CI checks to finish and resolve failures before merging.
6. Merge through the pull request only. Do not fast-forward, merge locally into `main`, or push commits directly to `main`. An instruction to merge a change means merge its PR after checks pass.
7. After the PR is merged, switch to `main`, pull the merged changes, and delete the topic branch locally and remotely if appropriate.

If a PR cannot be opened or its checks cannot be completed, report the blocker and leave the change on its topic branch. Do not bypass the PR workflow to finish the merge.
