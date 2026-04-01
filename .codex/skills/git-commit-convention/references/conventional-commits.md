# Conventional Commits Reference

Source:
- https://www.conventionalcommits.org/zh-hans/v1.0.0/

## Official Core Rules

The official 1.0.0 structure is:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Key rules from the specification:
- `feat` means a new feature and maps to a SemVer `MINOR` change.
- `fix` means a bug fix and maps to a SemVer `PATCH` change.
- Breaking changes require a `BREAKING CHANGE: ...` footer, and may also use `!` after `type` or `type(scope)`.
- Other types are allowed, including common Angular-style types such as `build`, `chore`, `ci`, `docs`, `style`, `refactor`, `perf`, and `test`.
- `scope` is optional and should provide extra context, for example `feat(parser): ...`.

## HomeStock Project Conventions

These conventions are repo-specific defaults layered on top of the official spec:

- Write commit subjects in English to match existing history.
- Prefer one logical change per commit.
- Prefer the nearest top-level area as the scope:
  - `config`
  - `logger`
  - `docs`
  - `internal`
  - `repo`
- Omit scope when the change spans multiple unrelated areas and cannot be split cleanly.
- Keep the subject concise and imperative.
- Do not add a trailing period to the subject.

## Type Selection Heuristics

- `feat`: add user-visible or developer-visible behavior
- `fix`: correct incorrect behavior or a defect
- `docs`: documentation-only changes
- `refactor`: restructure code without changing intended behavior
- `test`: add or adjust tests
- `perf`: improve performance characteristics
- `build`: change dependency or build tooling behavior
- `ci`: change CI workflow or automation pipeline
- `chore`: maintenance work that does not fit the types above
- `style`: formatting-only changes with no logic impact

## Repo Examples

- `feat(config): add JSON config loading and env overrides`
- `feat(logger): add third-party logger adapter`
- `docs(repo): add configuration example file`
- `test(config): cover invalid env override handling`

## Breaking Change Example

```text
feat(config)!: rename database type field to driver

BREAKING CHANGE: configuration files must use database.driver instead of database.type.
```
