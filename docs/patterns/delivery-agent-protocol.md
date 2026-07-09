<!-- SPDX-License-Identifier: Apache-2.0 -->
# Delivery-agent dispatch protocol

Shared execution contract for every Zynax delivery subagent defined under `.claude/agents/`
(go-services, python-adapters, bdd-contract, infra-helm, ci-release, spdd-canvas, post-merge).
Orchestrating commands (`/deliver`, `/lib:deliver-batch`, `/lib:deliver-one`) MUST NOT paste
this into dispatch prompts — each agent reads this file once at startup, right after its domain
guide at `.claude/commands/experts/<name>.md`.

> **Canonical home for env-constraint rules.** Rules that exist because of the subagent runtime
> (sandbox Bash limits, non-persistent shell state, literal-path discipline) belong HERE — never
> scattered into the expert guides. In the Session Learnings block (§9) such proposals carry
> `Category: env-constraint`; applying one means editing this file, not a guide.

---

## 1 — Dispatch inputs

The dispatcher provides, in your task prompt:

| Input | Meaning |
|---|---|
| `ISSUE` | Story issue number + title (or PR number + merge SHA for post-merge) |
| `REPO` | Absolute path of the main checkout |
| `WT` | Your **literal, private** worktree path (e.g. `/tmp/zynax-orch-<run>-<N>`) |
| Context files | 2–3 repo paths named by the issue body or canvas O-step |

Never invent a worktree path; never work in `REPO` directly — the user's checkout must not be
mutated.

## 2 — Sandbox Bash discipline (read before your first command)

You run under a Bash sandbox that allows SINGLE commands but DENIES compound/chained forms, and
shell state does NOT persist between Bash calls. Concretely:

- NEVER chain: no `cd dir && ...`, no `a; b`, no `until/do/done` loops, no pipes between
  commands, no `env VAR=x cmd` prefix, and no call mixing `rm -rf` / `worktree remove --force`
  with another operation. Each of these is reliably denied.
- NEVER `cd` and NEVER rely on shell variables across calls. Use LITERAL paths with
  `git -C <path> ...`, `GOWORK=off go -C <path> ...`, and `make -C <path> <target>`
  (a bare `GOWORK=off` env assignment prefix is fine; an `env` prefix is not).
- Multiline commit messages: write the message to a file and `git commit -s -F <file>`
  (`-m` with embedded newlines is denied).
- Wait for CI with `gh pr checks <PR> --watch --interval 30` (foreground; never a sleep-poll
  loop or a background watch — ending your turn "to wait" strands the delivery).
- On any denial: split into single commands and retry — concluding "Bash is denied" and
  abandoning the task is wrong.
- Read the active milestone with a single call: `bash <REPO>/automation/milestone-env.sh`
  (prints shell-quoted `KEY=value` lines; parse the output — do not `eval` in sandbox mode).

## 3 — Isolated worktree (your FIRST actions; one Bash call each)

```bash
git -C <REPO> worktree remove <WT> --force   # ignore an error if absent
git -C <REPO> fetch origin --prune
git -C <REPO> worktree add <WT> origin/main
```

`<WT>` is yours alone: no defensive `git checkout`, no branch verification before each call,
and do NOT avoid `git add` — reference the tree by literal path in every command.

## 4 — Claim (deterministic key — the sole cross-session mutex)

1. Confirm the issue is still OPEN and not labelled `status: in-progress` by another session.
   If already claimed: remove your worktree (§7), stop, report "claim lost".
2. The claim branch ref is `<type>/<N>` — a pure function of the issue number, NO slug.
   Push it EMPTY before any code (`git -C <WT> push -u origin <type>/<N>`); only one push wins.
   A rejected push means the story is taken → stop, clean up, report "claim lost".
3. Apply a human-readable slug only AFTER the claim push wins (rename + push the slugged ref,
   delete the bare ref); a slug must never enter the claim push.

## 5 — Implement, gate, commit, PR

- Engineering rules are NOT restated here: read `AGENTS.md` (constitution) and `CLAUDE.md`
  (dev loop, PR size, SPDD) sections relevant to your change. `GOWORK=off` for every `go`
  command inside `services/*/`, `cmd/zynax/`, `protos/tests/`.
- Run all local gates before committing; never commit on a red gate.
- **Runtime smoke — required when the diff touches a runtime path** (`infra/docker-compose/**`,
  `infra/helm/**`, a Makefile `demo`/`run-local`/compose target, `services/*/cmd/**`,
  `agents/adapters/**`, `cmd/zynax*`): start from a clean slate (down `-v`), boot the issue's
  DOCUMENTED path, fail on any Exited/unhealthy container, then run the documented path a
  SECOND time on the same volumes (persistence bugs surface on run #2). Config render, build,
  or CI green is NOT runtime evidence — map every "runs"/"demo"/"end-to-end" acceptance
  criterion to an actual execution with captured output.
- **Reconcile ALL status surfaces in the same diff**, driven by live issue/PR state: flip this
  story's row ⬜→✅ in the active planning doc; update `state/current-milestone.md` (progress +
  "as of" date); mark the canvas O-step ✅ (`feat:` — and flip canvas `Status:` to
  `Implemented` if this closed the EPIC's last O-step, running `/lib:spdd-sync` if the
  implementation diverged); on EPIC completion or a service status change, also update the
  milestone/service tables in `README.md`, `ROADMAP.md`, `ARCHITECTURE.md`, `CLAUDE.md`; touch
  `services/<svc>/AGENTS.md` only if a new gRPC method, K8s resource type, or env var landed.
- Commit: `<type>(<scope>): <subject>` ≤72 chars, DCO `-s`, and an
  `Assisted-by: Claude/<model-id-of-this-session>` trailer (never Co-Authored-By for AI).
- PR body: build it from the canonical template in `docs/contributing/pr-templates.md`
  (your commit type's variant). Required sections, in order: `Closes #<N>`, **Why**,
  **What you'll get**, **Scope & boundaries**, **Test plan & acceptance** (one row per
  acceptance criterion with the exact verify command + result), **Evidence**,
  **Risk & rollback**, **Review aids**. For `feat:` add the SPDD line (canvas Aligned +
  security-review PASS). Leave the "Post-merge digest sync → main" Evidence line as a
  placeholder — the post-merge verifier fills it.
- Never write a literal skip-ci token (`[skip` + `ci]` and variants) in a commit message or PR
  body — write "skip-ci marker" to refer to it.

## 6 — CI wait + merge queue (ADR-047)

- `gh pr checks <PR> --watch --interval 30` in the foreground. All *required* checks green is
  the gate; `UNSTABLE` with only advisory checks pending is ready.
- Arm the queue: `gh pr merge <PR> --squash --auto` (never `--rebase`; never `--merge`).
- `BEHIND` needs NO action — the queue validates against current main; a force-push ejects a
  queued PR. Only `DIRTY` (real conflicts) needs `git rebase --signoff origin/main` +
  `git push --force-with-lease` (SSH signing is preserved via `rebase.gpgSign`), then re-arm.
- *Fallback (no merge-queue rule on main — pre-cutover or rollback):* rebase onto
  `origin/main` + `--force-with-lease`, re-validate, then merge, as before ADR-047.
- You are not done until the merge state is reported — never end your turn mid-wait.

## 7 — Cleanup (your LAST action, success or failure)

```bash
git -C <REPO> worktree remove <WT> --force
```

A single Bash call, literal path, no chained `rm -rf`. A lingering directory (root-owned Docker
caches) is harmless — the orchestrator's sweep reclaims the path.

## 8 — Result format (required — the orchestrator parses this)

```
## Result
- Issue: #NNN
- PR: #NNN
- Merge SHA: <full sha of squash merge commit on main, or "not merged">
- CI: green / red / pending
- Affected services: <comma-separated list or "none">
```

## 9 — Session Learnings (required — emit verbatim in this shape so /learn can parse it)

```
## Session Learnings
- domain: <go-services|ci-release|infra-helm|python-adapters|bdd-contract|spdd-canvas>
- issue: #NNN
- date: YYYY-MM-DD

### Effective patterns
- <pattern>: <why it worked>

### Edge cases discovered
- <what>: <resolution>

### Failed approaches
- <what>: <why it failed>

### Proposed expert prompt update
- Rule: <exact text>
  Category: domain | structural-workaround | env-constraint
  Reason: <why permanent — for structural-workaround, name the shared-tree problem it works around>
```

The `post-merge` agent emits `domain: ci-release` (its learnings live in
`docs/ai-learnings/ci-release.md` — there is no separate post-merge learnings file).

Category guide: `domain` = genuine engineering knowledge (API shapes, proto fields, test
patterns) → expert guide. `env-constraint` = subagent-runtime constraint → THIS file.
`structural-workaround` = only exists to survive a shared working tree → rejected by default
(worktree isolation removed the root cause).

## 10 — Context discipline

Read only: your expert guide, this protocol, the issue body, and the context files named in the
dispatch. Never read files outside the issue scope. As a sizing guide: an implementation run's
instructions + initial reads should stay lean (on the order of ~12K tokens before coding; a
post-merge verification ~20K total) — a run that balloons past that is reading outside scope.
If you notice your context has been compacted mid-run, finish the current step, stop at the
next safe boundary, and emit your guide's split report (branch, files written, tests state,
resume point) instead of pressing on.
