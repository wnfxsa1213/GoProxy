# Coding Style Guide

> 此文件定义团队编码规范，所有 LLM 工具在修改代码时必须遵守。
> 提交到 Git，团队共享。

## General
- Prefer small, reviewable changes; avoid unrelated refactors.
- Keep functions short (<50 lines); avoid deep nesting (≤3 levels).
- Name things explicitly; no single-letter variables except loop counters.
- Handle errors explicitly; never swallow errors silently.

## Language-Specific

### Go
- Follow standard Go conventions (`gofmt`, `go vet`).
- Log messages use `[module]` prefix: `[pool]`, `[fetch]`, `[health]`, `[optimize]`, `[monitor]`, `[socks5]`, `[proxy]`, `[tunnel]`, `[storage]`, `[source]`.
- Comments and log messages in Chinese.
- Quality grades: S (<=500ms), A (501-1000ms), B (1001-2000ms), C (>2000ms).
- CGO_ENABLED=1 required (sqlite3 dependency).

## Git Commits
- Conventional Commits, imperative mood.
- Atomic commits: one logical change per commit.

## Testing
- No Go unit tests in this project; validate via integration test scripts in `test/`.
- Test against a running instance using shell/Go/Python scripts.

## Security
- Never log secrets (tokens/keys/cookies/JWT).
- Validate inputs at trust boundaries.
- Proxy auth credentials handled via environment variables only.
