# Learnings: Python Adapter Engineer

> Format: each entry has `Seen in:` (issue/session) and `Date:` (YYYY-MM-DD).
> Updated by `/m6-learn` after each batch. Human-reviewed before merging.

---

## Effective patterns

- **Always choose the Adapter path for wrapping existing systems — never SDK for integration work.**
  The SDK is for building net-new agents. Using the SDK to wrap a LangGraph app creates
  unnecessary coupling to the Zynax agent lifecycle. Adapters are translation layers only.
  Seen in: M5 adapter library (http/git/ci/llm/langgraph adapters). Date: 2026-06-06.

- **`Pydantic BaseSettings` with an `env_prefix` is the standard config pattern.**
  All 5 existing adapters use this. It's consistent, testable, and avoids hardcoding.
  The prefix prevents collisions with other env vars in CI.
  Seen in: agents/adapters/* (all 5 adapters). Date: 2026-06-06.

- **`asyncio.gather(*coroutines)` for parallel downstream calls — never sequential `await` in a loop.**
  If an adapter calls multiple downstream services, parallel gather reduces latency proportionally.
  Seen in: M5 llm-adapter (#383). Date: 2026-06-06.

- **`bandit -ll` (low-and-above threshold) before every commit.**
  Running bandit without flags only reports MEDIUM+ by default, missing LOW severity issues
  that may accumulate. The `-ll` flag (report LOW and above) catches them early.
  Seen in: M5.D security baseline #461. Date: 2026-06-06.

---

## Edge cases discovered

- **gRPC `aio.insecure_channel` must be closed explicitly — not garbage-collected.**
  In async Python, the event loop closes before `__del__` runs, causing "Task destroyed but
  it is pending" warnings in tests. Always close in a `finally` block or async context manager.
  Seen in: M5 agent SDK design. Date: 2026-06-06.

- **`pip-audit` fails on packages with yanked releases even when not directly imported.**
  Transitive dependencies can pull in yanked versions. Pin all direct deps to a known-good
  version in `requirements.txt` and run `pip-audit` in CI to catch regressions.
  Seen in: M5.D security audit. Date: 2026-06-06.

- **`mypy --strict` fails on `grpc.aio` stub types — they're incomplete in the current stubs package.**
  Suppress with `# type: ignore[attr-defined]` only for gRPC stubs, not for business logic.
  Add a comment explaining the suppression.
  Seen in: M5 sdk type annotation work. Date: 2026-06-06.

---

## Failed approaches

- **Using `requests` (sync) inside an async adapter handler.**
  Blocks the event loop for the duration of the HTTP call, starving other concurrent requests.
  Always use `httpx.AsyncClient` for HTTP inside async code.
  Seen in: Early CI adapter draft. Date: 2026-06-06.

---

## Proposed expert prompt updates

*(none yet — populate after first batch of Python adapter expert sessions)*

## Session — 2026-06-08 (issue #805)

### Effective patterns

- **PyPI Trusted Publisher (OIDC) — use `continue-on-error: true` until registration.**
  The OIDC publisher must be manually registered at pypi.org/manage/account/publishing/ before
  the publish step can succeed. Any CI gate that calls `hatch publish` before registration
  will fail. Wrap with `continue-on-error: true` and add clear instructions in AGENTS.md.

- **`pip-audit` false positives on CI runners vs local.** `PYSEC-2026-196` affects pip itself —
  present in GitHub Actions-hosted runners (pip 24.x) but not in managed containers. Add
  `--ignore-vuln PYSEC-2026-196` to the CI step and document in `[tool.pip-audit]` section
  of `pyproject.toml` with a comment explaining the scope.

## Session — 2026-06-09 (issue #808)

### Effective patterns

- **Read `pyproject.toml` ruff config before judging compliance**: The `[tool.ruff.lint]` settings (select, extend-ignore, pydocstyle convention) determine what ruff actually enforces — always check before assessing what "passes". A file that passes `ruff check` may still fail `ruff check --select D` because the CLI `--select` resets `extend-ignore`. Verify both forms when the acceptance criteria specifies both. Seen in: #808.
- **Remove suppression rules when adding the missing docstring**: When a `D105` (magic method docstring) was suppressed via `extend-ignore`, the fix was to add the missing docstring AND remove the suppression — not just the suppression. The resulting config is cleaner and the lint gate is more honest. Seen in: #808.

### Edge cases discovered

- **Root-owned `.ruff_cache` from Docker blocks host pre-commit hook**: Running `make lint-python` via Docker creates `.ruff_cache` owned by root. The pre-commit ruff hook on the host then fails with a permission error. Resolution: prepend `RUFF_CACHE_DIR=/tmp/ruff-cache-<issue>` to the `git commit` command. Seen in: #808.
- **Parallel M6 activity causes `mergeStateStatus: BEHIND` loop**: Main accumulates commits from concurrent agents during CI runs. `gh pr merge --squash` without `--auto` fails repeatedly when the branch is BEHIND. Use `--auto` immediately after the first push so GitHub fires the merge once all checks pass on the latest HEAD. Seen in: #808.

### Proposed expert prompt update

- Rule: Before running `git commit` in a project with Docker-based lint tooling, check if `.ruff_cache` (or equivalent cache dirs) are owned by root. If so, prepend `RUFF_CACHE_DIR=/tmp/ruff-cache-<issue>` to the commit command to avoid pre-commit hook permission failures.
  Category: structural-workaround
  Reason: The tools Docker image creates root-owned cache dirs that block host pre-commit hooks — affects all Python-touching PRs in this repo.
