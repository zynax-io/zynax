<!-- SPDX-License-Identifier: Apache-2.0 -->

# Private Development Configuration

> This document describes the recommended approach for maintaining personal
> development configuration that **must not** be committed to the public
> `keel-io/keel` repository.
>
> This is for the project maintainer(s). Contributors using AI tools should
> see [`docs/ai-assistant-setup.md`](ai-assistant-setup.md) for the simpler
> contributor setup.

---

## The Problem

When developing Keel with an AI coding assistant, you accumulate configuration
that is personal to your development environment:

- AI tool project settings (`.claude/settings.json`, hooks, permissions)
- AI tool memory and session data
- Personal git hooks
- Local environment overrides

None of this belongs in the public repository. But it needs to be version-controlled
somewhere — losing it when you change machines or reinstall tools is painful.

---

## The Solution: `keel-dev-config` Private Repository

A private repository `<your-github-username>/keel-dev-config` holds everything
that should be tracked but not public.

```
keel-dev-config/           ← private git repo
  .claude/
    settings.json          ← Claude Code project permissions
    commands/              ← custom slash commands
  hooks/
    commit-msg             ← personal git hook additions
  notes/
    architecture-notes.md  ← your working notes (not docs/)
  setup.sh                 ← links everything into place
  README.md
```

---

## Setup

### Step 1 — Create the private repository

```bash
gh repo create <your-username>/keel-dev-config \
  --private \
  --description "Personal development configuration for keel-io/keel" \
  --clone
```

### Step 2 — Create the directory structure

```bash
cd keel-dev-config
mkdir -p .claude/commands hooks notes
```

### Step 3 — Add your Claude Code settings

```bash
cat > .claude/settings.json << 'EOF'
{
  "permissions": {
    "allow": [
      "Bash(make *)",
      "Bash(git *)",
      "Bash(go *)",
      "Bash(uv *)",
      "Bash(buf *)",
      "Bash(docker *)",
      "Bash(gh *)"
    ],
    "deny": [
      "Bash(rm -rf *)",
      "Bash(git push --force *)"
    ]
  }
}
EOF
```

### Step 4 — Write the setup script

```bash
cat > setup.sh << 'SETUP'
#!/usr/bin/env bash
# Links keel-dev-config into the keel project directory.
# Run from inside the keel/ project directory:
#   bash .dev/setup.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "Linking keel-dev-config into ${PROJECT_DIR}..."

# Claude Code project settings
mkdir -p "${PROJECT_DIR}/.claude"
ln -sf "${SCRIPT_DIR}/.claude/settings.json" \
        "${PROJECT_DIR}/.claude/settings.json"

# Custom slash commands (if any)
if [ -d "${SCRIPT_DIR}/.claude/commands" ]; then
  mkdir -p "${PROJECT_DIR}/.claude/commands"
  for cmd in "${SCRIPT_DIR}/.claude/commands"/*.md; do
    [ -f "$cmd" ] && \
      ln -sf "$cmd" "${PROJECT_DIR}/.claude/commands/$(basename "$cmd")"
  done
fi

# Personal git hooks (merged with existing)
HOOKS_DIR="${PROJECT_DIR}/.git/hooks"
for hook in "${SCRIPT_DIR}/hooks"/*; do
  [ -f "$hook" ] && \
    ln -sf "$hook" "${HOOKS_DIR}/$(basename "$hook")" && \
    chmod +x "${HOOKS_DIR}/$(basename "$hook")"
done

echo "Done. Personal config is active."
SETUP
chmod +x setup.sh
```

### Step 5 — Link into the project

Clone the dev config alongside your project and run the setup script:

```bash
# From inside the keel/ project directory
git clone git@github.com:<your-username>/keel-dev-config.git .dev
bash .dev/setup.sh
```

The `.dev/` directory is in `.gitignore` — it will never be accidentally pushed
to `keel-io/keel`.

### Step 6 — Commit and push the private config

```bash
cd .dev
git add .
git commit -m "chore: initial keel dev config"
git push
```

---

## What Lives Where

| File / Directory | Location | Committed to |
|-----------------|----------|-------------|
| `AGENTS.md` files | `keel/` project | `keel-io/keel` (public) |
| `.claude/settings.json` | `keel/.dev/.claude/` → symlinked | `<you>/keel-dev-config` (private) |
| `.claude/commands/` | `keel/.dev/.claude/commands/` → symlinked | `<you>/keel-dev-config` (private) |
| `.claude/` session data | `~/.claude/` (home dir) | nowhere — ephemeral |
| Memory files | `~/.claude/projects/.../memory/` | nowhere — local only |
| Personal git hooks | `keel/.dev/hooks/` → symlinked | `<you>/keel-dev-config` (private) |
| Working notes | `keel/.dev/notes/` | `<you>/keel-dev-config` (private) |

---

## Restoring on a New Machine

```bash
# Clone the public project
git clone git@github.com:keel-io/keel.git
cd keel

# Clone your private dev config into .dev/
git clone git@github.com:<your-username>/keel-dev-config.git .dev

# Wire everything up
bash .dev/setup.sh

# Done — all your settings are active
make bootstrap
make dev-up
```

---

## What NOT to Put in `keel-dev-config`

- No secrets, API keys, or tokens (even in a private repo)
- No production credentials
- No `.env` files with real values — use `.env.example` patterns
- No personal opinions about the project architecture — those go in GitHub Discussions

---

## When to Promote Config to the Public Repo

If you add something to `.dev/` that you later realise other contributors would
benefit from, promote it:

1. Move the file from `.dev/` to the appropriate location in `keel/`
2. Open a PR with the change
3. Remove the local copy from `.dev/` and update `setup.sh`

Examples of things that started as private and became public:
- An `AGENTS.md` section that was tested privately and refined
- A custom `make` target that saves time during development
- A `.claude/commands/` slash command that becomes universally useful
