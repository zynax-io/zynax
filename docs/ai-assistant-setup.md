<!-- SPDX-License-Identifier: Apache-2.0 -->

# AI Assistant Setup for Zynax Contributors

> This guide is for contributors who use AI coding assistants (Claude Code,
> Cursor, GitHub Copilot, Gemini Code Assist, or others) when working on Zynax.
>
> **AI assistance is entirely optional.** Nothing in this guide is required to
> contribute. The project has no dependency on any AI tool.

---

## How Zynax Uses `AGENTS.md`

Zynax provides `AGENTS.md` files at the root and in each major directory:

```
zynax/
  AGENTS.md                    ← root engineering contract
  services/AGENTS.md           ← Go service patterns
  services/<service>/AGENTS.md ← per-service rules
  agents/AGENTS.md             ← Python adapter patterns
  protos/AGENTS.md             ← proto contract rules
  spec/AGENTS.md               ← YAML schema rules
  infra/AGENTS.md              ← infrastructure rules
```

These files are **engineering documentation** — they describe the architecture,
standards, and constraints of the project. They are written for human contributors
first and happen to also be read by AI coding assistants as configuration.

`AGENTS.md` is an emerging cross-tool standard. All major AI coding tools
read these files automatically when present.

---

## Tool-Specific Setup

### Claude Code (Anthropic)

Claude Code reads `AGENTS.md` natively. No configuration needed beyond cloning
the repository — the engineering contracts are automatically active.

For personal settings that should **not** be committed to the repository
(hooks, permissions, custom commands), create a local `.claude/` directory:

```bash
# This directory is in .gitignore — safe to create
mkdir -p .claude

# Optional: project-specific settings
cat > .claude/settings.json << 'EOF'
{
  "permissions": {
    "allow": [
      "Bash(make *)",
      "Bash(git *)",
      "Bash(go *)",
      "Bash(uv *)",
      "Bash(buf *)"
    ]
  }
}
EOF
```

The `.claude/` directory is gitignored. Its contents are yours and will never
be pushed to the repository.

See [`docs/dev-config-private.md`](dev-config-private.md) for the recommended
private configuration repository approach.

### Cursor

Cursor reads `AGENTS.md` automatically as project rules. No setup needed.

For personal rules that supplement `AGENTS.md` without being committed:
```bash
# .cursor/ is gitignored
mkdir -p .cursor
# Add personal rules in .cursor/rules if desired
```

### GitHub Copilot

Copilot reads `AGENTS.md` in supported editors (VS Code, JetBrains).
Enable it under: Settings → GitHub Copilot → Use AGENTS.md for context.

### Other Tools

If your tool supports a custom instructions file, point it to `AGENTS.md`
at the root and the relevant layer `AGENTS.md` for the directory you are
working in.

---

## What the `AGENTS.md` Files Cover

Reading these files before working in a section of the codebase will save
you significant rework. Key things they document:

| File | What it covers |
|------|---------------|
| `AGENTS.md` | Three-layer architecture, workflow model, capability model, hard constraints |
| `services/AGENTS.md` | Go service structure, domain/api/infra layers, error handling patterns |
| `services/<svc>/AGENTS.md` | Domain model, RPC methods, specific rules for that service |
| `agents/AGENTS.md` | Python adapter pattern, gRPC stub usage, runtime protocol |
| `protos/AGENTS.md` | Proto naming conventions, backward-compatibility rules |
| `spec/AGENTS.md` | YAML manifest schemas, validation rules |
| `infra/AGENTS.md` | Docker, Helm, environment variable conventions |

---

## AI Contribution Labelling

If you used an AI assistant to generate substantial portions of a PR:

1. Add the `ai-assisted` label to the PR.
2. Include in the PR description:
   ```
   AI assistance: <tool name> / <model> (what it helped with)
   ```
3. Add `Assisted-by:` to the squash commit footer:
   ```
   Assisted-by: Claude Code/claude-sonnet-4-6
   ```
   **Do not use `Co-Authored-By:` or `Signed-off-by:` for AI tools.** Those tags
   are reserved for humans certifying the Developer Certificate of Origin. Adding
   an AI tool there misrepresents who certified the DCO.

4. If your tool appends `Co-Authored-By: Claude ...` automatically (Claude Code does
   this), remove that line before pushing and replace it with `Assisted-by:`.

This is informational. It does not change the review requirements or quality bar.
The human author is fully responsible for all code regardless of how it was generated.

See `CONTRIBUTING.md §11` for the complete AI contribution policy.

---

## What AI Tools Must Not Do

These constraints from `AGENTS.md §12` apply to AI-generated output as strictly
as to hand-written code. Review AI output against these before committing:

- No `panic` in production Go code
- No `print()` in Python — use `structlog`
- No secrets or credentials anywhere in the codebase
- No cross-service imports (a service's `internal/` is private)
- No hardcoded engine or LLM model names — env vars only
- No changes outside the scope of the current issue
- The `.feature` file must exist before any implementation — AI cannot skip this
