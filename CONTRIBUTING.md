# Contributing to kflow

Thank you for your interest in contributing to kflow! This document explains how to get involved.

---

## Getting Started

1. **Fork the repository** on GitHub — click the "Fork" button on the top right of the repository page.

2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/<your-username>/kflow.git
   cd kflow
   ```

3. **Add the upstream remote** so you can pull future changes:
   ```bash
   git remote add upstream https://github.com/pastorenue/kflow.git
   ```

---

## Workflow

1. **Sync with upstream** before starting any work:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Create a feature branch** off `main`:
   ```bash
   git checkout -b feat/your-feature-name
   ```
   Use a descriptive prefix: `feat/`, `fix/`, `docs/`, `refactor/`, `test/`.

3. **Make your changes.** Keep commits focused and atomic.

4. **Run tests and linters** before pushing:
   ```bash
   go build ./...
   go test ./...
   ```

5. **Push your branch** to your fork:
   ```bash
   git push origin feat/your-feature-name
   ```

6. **Open a Pull Request** from your fork's branch to `upstream/main` on GitHub.

---

## Pull Request Guidelines

- **One concern per PR.** Avoid bundling unrelated changes.
- **Reference any related issue** in the PR description (e.g., `Closes #42`).
- **Write a clear description** of what the change does and why.
- **Include tests** for any new behaviour or bug fix.
- **Keep the diff small** when possible — smaller PRs are reviewed faster.
- All PRs must pass CI before they will be merged.

---

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep comments minimal — only where logic is non-obvious.
- Avoid over-engineering. The right amount of complexity is the minimum needed for the task.
- Do not introduce security vulnerabilities (SQL injection, command injection, hardcoded secrets, etc.). See the Security section in [CLAUDE.md](CLAUDE.md) for the full list of rules.

---

## Reporting Issues

Open a GitHub Issue with:
- A clear title and description.
- Steps to reproduce (for bugs).
- Expected vs. actual behaviour.
- Relevant logs or error messages.

---

## Architecture

Before implementing any package, read the relevant phase file in `docs/phases/` and the top-level [AGENTS.md](AGENTS.md). The phase files are the authoritative specification.

---

## Questions

Open a GitHub Discussion or comment on the relevant issue. We're happy to help you get started.
