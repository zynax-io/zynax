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
