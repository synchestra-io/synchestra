# Why there is no README.md here

GitHub treats `.github/README.md` specially: if it exists, GitHub displays it
on the repository's main page **instead of** the root `README.md`.

Because this project's root `README.md` is the canonical entry point for both
humans and agents, a `.github/README.md` must never be committed.

The [`.github/hooks/pre-commit`](hooks/pre-commit) hook enforces this rule automatically.
