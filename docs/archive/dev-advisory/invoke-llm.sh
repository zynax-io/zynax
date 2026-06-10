#!/usr/bin/env bash
# LLM invocation wrapper — swappable provider backend.
# Usage: invoke-llm.sh <system-prompt-file> [input-file|-]
# Env: LLM_PROVIDER (claude|ollama|openai-compat), LLM_MODEL,
#      OLLAMA_BASE_URL, ANTHROPIC_API_KEY, OPENAI_API_KEY, OPENAI_BASE_URL

set -euo pipefail

SYSTEM_FILE="${1:?Usage: invoke-llm.sh <system-prompt-file> [input-file]}"
INPUT_FILE="${2:--}"

SYSTEM=$(cat "$SYSTEM_FILE")
INPUT=$(cat "$INPUT_FILE")
PROVIDER="${LLM_PROVIDER:-claude}"

case "$PROVIDER" in
  claude)
    MODEL="${LLM_MODEL:-claude-sonnet-4-6}"
    curl -sf https://api.anthropic.com/v1/messages \
      -H "x-api-key: ${ANTHROPIC_API_KEY:?ANTHROPIC_API_KEY required for claude provider}" \
      -H "anthropic-version: 2023-06-01" \
      -H "Content-Type: application/json" \
      -d "$(jq -nc \
        --arg m "$MODEL" --arg s "$SYSTEM" --arg u "$INPUT" \
        '{model:$m, max_tokens:4096, system:$s,
          messages:[{role:"user",content:$u}]}'
      )" | jq -r '.content[0].text'
    ;;

  ollama)
    MODEL="${LLM_MODEL:-qwen2.5-coder:1.5b}"
    BASE="${OLLAMA_BASE_URL:-http://localhost:11434}"
    curl -sf "${BASE}/api/chat" \
      -H "Content-Type: application/json" \
      -d "$(jq -nc \
        --arg m "$MODEL" --arg s "$SYSTEM" --arg u "$INPUT" \
        '{model:$m, stream:false,
          messages:[{role:"system",content:$s},{role:"user",content:$u}]}'
      )" | jq -r '.message.content'
    ;;

  openai-compat)
    MODEL="${LLM_MODEL:?LLM_MODEL required for openai-compat provider}"
    BASE="${OPENAI_BASE_URL:-http://localhost:8000}"
    curl -sf "${BASE}/v1/chat/completions" \
      -H "Authorization: Bearer ${OPENAI_API_KEY:-none}" \
      -H "Content-Type: application/json" \
      -d "$(jq -nc \
        --arg m "$MODEL" --arg s "$SYSTEM" --arg u "$INPUT" \
        '{model:$m,
          messages:[{role:"system",content:$s},{role:"user",content:$u}]}'
      )" | jq -r '.choices[0].message.content'
    ;;

  *)
    echo "Unknown LLM_PROVIDER: $PROVIDER (expected: claude|ollama|openai-compat)" >&2
    exit 1
    ;;
esac
