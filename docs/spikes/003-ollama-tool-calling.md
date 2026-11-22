# Spike-003: Local Tool Calling Reliability with Ollama (Qwen 2.5 Coder 7B vs. 14B)

- **Status:** `Pending`
- **Date:** `2026-11-21`
- **Timebox:** `1 day`
- **Informs / Blocks:** [ADR-004](../adr/004-llm-provider-abstraction.md), [Benchmarks](../benchmarks.md), [Requirements](../requirements.md)

## Objective & Question

**Objective:** Test structured tool calling (`read_file`, `apply_patch`) over Ollama's OpenAI-compatible `v1/chat/completions` endpoint using a Go harness.

**Core Question:** What JSON tool-calling formatting success rate, schema adherence, and latency do `Qwen 2.5 Coder 7B` vs. `14B` achieve running locally under Ollama?

## Context

Graft's Agent Runtime relies on deterministic tool invocations via MCP ([RFC-001 Section 3 & 5](../architecture.md)) and aims to support local Ollama models seamlessly alongside cloud APIs ([ADR-004](../adr/004-llm-provider-abstraction.md)). Smaller local models often struggle with strict JSON schema adherence, escaped multi-line unified diffs inside JSON strings, or multi-turn tool-result loops.

## Experiment Plan

1. Define JSON tool schemas mirroring Graft's core MCP tools (`read_file`, `list_symbols`, `apply_patch`).
2. Run a suite of 15–20 synthetic single-turn and multi-turn coding prompts against Ollama (`v1/chat/completions` with `tools` parameter):
   - `qwen2.5-coder:7b` (pinned tag)
   - `qwen2.5-coder:14b` (pinned tag)
3. Measure:
   - Valid `tool_calls` JSON parse rate.
   - Schema validity (correct tool name + required arguments + valid unified diff syntax).
   - Tokens/sec and end-to-end turn latency on local hardware.
4. Test whether prompt-level fallback parsing or strict JSON mode improves recovery when `tool_calls` formatting fails.

## Evaluation Criteria

- Quantitative comparison (`%` valid tool calls, `%` valid unified diffs, latency) between 7B and 14B models.
- Clear threshold determining whether native `tools` support in Ollama is production-ready for Graft or requires retry/repair prompts in the Agent Runtime.

## Result

`Pending`

## Key Learnings

- `TBD` — Expected observations to capture:
  - Ollama version and quantized model tags tested.
  - Common failure modes (e.g., JSON escaping errors inside `apply_patch`, hallucinated tool arguments, returning markdown instead of `tool_calls`).
  - Hardware VRAM usage and context-window limits (`num_ctx`).

## Impact on Requirements & ADRs

- **Requirements ([requirements.md](../requirements.md)):** Will define tool-call validation/retry rules inside the agent loop and minimum local hardware/model assumptions.
- **ADRs & Benchmarks:** Will validate [ADR-004](../adr/004-llm-provider-abstraction.md) and populate baseline local configuration metadata in [benchmarks.md](../benchmarks.md).

## Recommendation & Next Steps

- `TBD`
