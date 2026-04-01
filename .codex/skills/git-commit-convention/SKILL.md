---
name: git-commit-convention
description: Draft and review Git commit messages for the HomeStock repository using Conventional Commits. Use when Codex needs to write, revise, or validate commit messages, choose a commit type/scope, split changes into coherent commits, or explain why a commit message does not match the project's commit convention.
---

# Git Commit Convention

Inspect the staged diff and recent commits before drafting a message. Base the message on the actual change, not only the user's wording.

## Workflow

1. Determine whether the change is one logical commit.
If the diff mixes unrelated concerns, recommend splitting before drafting a message.

2. Choose the Conventional Commits type.
Prefer `feat` for new behavior and `fix` for bug fixes.
Use `docs`, `refactor`, `test`, `chore`, `build`, `ci`, `perf`, or `style` when they describe the change more precisely.

3. Choose the scope from the narrowest subsystem that best matches the diff.
Prefer a single scope.
Omit the scope only when the change is truly repo-wide and no single subsystem is dominant.

4. Write the subject as:
`<type>[optional scope]: <description>`

5. Keep the subject short, specific, and in imperative English.
Use lowercase for `type` and `scope`.
Do not end the subject with a period.

6. Add a body only when the reason, tradeoff, or migration detail is important.

7. Mark breaking changes with both:
`<type>(<scope>)!:` in the subject
`BREAKING CHANGE: ...` in the footer

## HomeStock Scope Rules

Choose the nearest affected area:
- `config`: `pkg/config`, config examples, config loading behavior
- `logger`: `pkg/logger`
- `docs`: `.docs`, `README.md`, other documentation-only edits
- `internal`: `internal/` packages
- `repo`: root-level repo wiring such as `.gitignore`, module metadata, shared project setup

If one subsystem clearly dominates, use that scope even if a few support files also changed.
If multiple subsystems are equally important, prefer splitting commits. If splitting is not practical, omit the scope instead of inventing a vague one.

## Output Rules

When the user asks for a commit message, return:
- one primary commit message
- an optional short rationale only if the type or scope is non-obvious

When the user asks to validate an existing message:
- state whether it matches the convention
- give the corrected message if it does not

When the user asks for a commit after code changes:
- inspect `git diff --cached` first
- if nothing is staged, inspect `git diff`
- draft the message from the actual changes

## Examples

- `feat(config): add JSON config loading and env overrides`
- `fix(logger): avoid leaking caller temp variable`
- `docs(repo): document configuration example file`
- `refactor(logger): simplify third-party logger adapter`
- `feat(config)!: rename database type field to driver`

For official rule details and repo-specific message selection guidance, read [references/conventional-commits.md](references/conventional-commits.md).
