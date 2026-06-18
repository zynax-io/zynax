<!-- SPDX-License-Identifier: Apache-2.0 -->

# Terminal casts

This directory holds the [asciinema](https://asciinema.org) terminal recordings
embedded across the docs and the top of the [README](../../README.md).

## `make-demo.cast` — needs a human recording

The README hero embed points at a **placeholder**. The real cast must be recorded
by a maintainer running the demo on real hardware, because the demo pulls a
multi-GB Ollama model and runs a live LLM inference — neither of which can be
recorded on the CI host.

### How to record it

```bash
# 1. One-time: pull the demo model (~2 GB) on the host.
ollama pull qwen2.5-coder:3b

# 2. Install the CLI if you have not already.
make install-cli            # → ~/bin/zynax (ensure ~/bin is on your PATH)

# 3. Record the one-command demo.
asciinema rec docs/casts/make-demo.cast --command "make demo"

# 4. Tear down the stack.
make demo-clean
```

### Publishing the embed

1. Upload the cast: `asciinema upload docs/casts/make-demo.cast`.
2. Copy the returned asciinema.org id.
3. In [`README.md`](../../README.md), replace both `PLACEHOLDER` tokens in the
   hero `[![asciicast …]]` line with that id, and delete the placeholder note
   directly beneath it.

Keep the raw `.cast` file committed here so the embed survives even if the
upload is ever removed upstream.
