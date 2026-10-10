# vLLM Serve Flags — Governance Plane Reference

> Architecture context: `user → ingress (orcn) → vLLM inference worker`
>
> vLLM is **never** directly user-facing. All traffic is internal between the ingress server and the
> worker. This means we want **maximum data richness**, **zero auth overhead on the wire**, and
> **fast/efficient transport**. Flags below are evaluated through that lens.

Legend:
- ✅ **ACTIVE** — already emitted in code (`text_generation.go` / `embedding.go` / `score.go`)
- 🔜 **CANDIDATE** — not in code yet, worth adding soon
- 📋 **FUTURE** — architecturally relevant but not needed now (e.g. gRPC migration)
- ❌ **NOT FOR US** — documents why we skip it

---

## 1. Observability & Metrics

| Flag | Status | Why it matters for us |
|---|---|---|
| `--enable-prompt-tokens-details` | ✅ ACTIVE (always) | Breaks down prompt token counts per request in the response. The ingress uses this for per-request cost accounting and billing attribution without a separate tokenizer call. |
| `--enable-per-request-metrics` | ✅ ACTIVE (always) | Populates vLLM's `/metrics` Prometheus endpoint with per-request histograms (TTFT, ITL, e2e latency). Not in response body — scrape via Prometheus. |
| `--disable-log-stats` | ✅ ACTIVE (prod only) | Silences the periodic `Stats: Avg throughput: ...` stdout dump. Redundant in prod since we scrape `/metrics`. In dev this is kept off so you can eyeball throughput. |
| `--collect-detailed-traces` | 🔜 CANDIDATE | Enables OpenTelemetry span collection. Pair with `--otlp-traces-endpoint` to feed our tracing backend. Useful when debugging latency spikes in the ingress→worker hop. |
| `--otlp-traces-endpoint` | 🔜 CANDIDATE | OTLP collector URL. Set when adding OTel tracing. |

---

## 2. Transport & Protocol

| Flag | Status | Why it matters for us |
|---|---|---|
| `--host` | ✅ ACTIVE (implicit `0.0.0.0`) | Binds to all interfaces inside the container so ingress can reach it. |
| `--port` | ✅ ACTIVE (`9000`) | Fixed port for predictable container networking. |
| `--uvicorn-log-level` | ✅ ACTIVE (`info` dev / `warning` prod) | In dev: `info` keeps `POST /v1/chat/completions 200 OK` lines visible — useful for debugging. In prod: `warning` silences HTTP access logs; errors/4xx/5xx still surface. Controlled by `APP_ENV`. |
| `--disable-fastapi-docs` | ❌ NOT FOR US | Not a valid vLLM CLI flag — docs are disabled via FastAPI config in application code, not CLI. |
| `--grpc` / `--grpc-port` | ❌ NOT FOR US | Replaces the HTTP server entirely — breaks OpenAI API compatibility. Would require full ingress client rewrite. Stay on HTTP. |
| `--grpc-max-recv-msg-size` | ❌ NOT FOR US | Tied to `--grpc`, which we are not using. |

---

## 3. Batching & Throughput

| Flag | Status | Why it matters for us |
|---|---|---|
| `--max-num-seqs` | ✅ ACTIVE (all tasks) | Caps concurrent sequences. The ingress queues overflow at this limit so we need to tune it per worker tier. Now wired for text gen too (was embedding/score only). |
| `--max-num-batched-tokens` | ✅ ACTIVE (all tasks) | Max tokens per scheduler batch. Tune alongside `max-num-seqs` for throughput vs. latency. Now wired for text gen too. |
| `--scheduler-delay-factor` | 🔜 CANDIDATE | Adds a small delay before scheduling to allow more requests to batch together. Helps throughput at the cost of slight latency. |
| `--preemption-mode` | 🔜 CANDIDATE | `swap` vs `recompute`. For our ingress pattern where requests can be retried, `recompute` saves GPU memory at the cost of latency on preemption. |

---

## 4. Memory & Performance

| Flag | Status | Why it matters for us |
|---|---|---|
| `--gpu-memory-utilization` | ✅ ACTIVE (all tasks, default `0.80`) | Reserves 20% GPU memory headroom. Configurable per deployment. |
| `--cpu-offload-gb` | ✅ ACTIVE (text gen, default `0`) | Allows model weight offload to CPU RAM. Useful for large models on smaller GPUs. |
| `--enable-prefix-caching` | ✅ ACTIVE (text gen, default `true`) | Caches KV for shared prefixes (system prompts). Major win for our use case where many requests share a system prompt via the ingress. |
| `--kv-cache-dtype` | ✅ ACTIVE (text gen, `fp8` optional) | FP8 KV cache halves cache memory. Trade-off: slight quality degradation. |
| `--swap-space` | 🔜 CANDIDATE | CPU swap space for KV cache (GB). Useful when prefix caching is aggressive. Default `4`. |
| `--block-size` | 🔜 CANDIDATE | KV cache block size in tokens. Larger blocks = less fragmentation but more waste per partially-used block. Default `16`. |
| `--enforce-eager` | ✅ ACTIVE (text gen / embedding, configurable) | Disables CUDA graph capture. Use for compatibility debugging only — CUDA graphs are faster in prod. |

---

## 5. Model Loading

| Flag | Status | Why it matters for us |
|---|---|---|
| `--dtype` | ✅ ACTIVE (`auto` for text gen, configurable for embedding/score) | `auto` picks bf16/fp16 based on GPU. The governance plane should surface this to the user so they can override for specific hardware. |
| `--tensor-parallel-size` | ✅ ACTIVE (all tasks, configurable) | Shards the model across N GPUs. The ingress must route all requests for a single model to the same worker group. |
| `--trust-remote-code` | ✅ ACTIVE (all tasks) | Required for custom model architectures on HuggingFace. We trust because we control which model IDs are deployable. |
| `--max-model-len` | ✅ ACTIVE (all tasks, optional) | Override the model's native max context. Useful to cap memory usage for large-context models. |
| `--load-format` | 🔜 CANDIDATE | `safetensors` (default), `pt`, `auto`. Could be exposed for advanced users deploying non-standard checkpoints. |
| `--quantization` | 🔜 CANDIDATE | `awq`, `gptq`, `squeezellm`, etc. Important for cost optimization on smaller GPUs. The governance plane should auto-detect from model config (many HF models ship quantized). |
| `--speculative-model` | 📋 FUTURE | Draft model for speculative decoding. Can 2–3× throughput for text gen. Worth adding once we have multi-model routing in the ingress. |

---

## 6. API Compatibility

| Flag | Status | Why it matters for us |
|---|---|---|
| `--served-model-name` | ✅ ACTIVE (all tasks) | Sets the model name in OpenAI-compatible responses. The ingress uses this to route `/v1/models` correctly. |
| `--api-key` | ✅ ACTIVE (all tasks, optional) | Internal API key between ingress and worker. Zero-trust internal networking: ingress authenticates to worker. |
| `--enable-auto-tool-choice` + `--tool-call-parser` | ✅ ACTIVE (text gen, configurable) | Required for function-calling models. The governance plane exposes this as the `tool_parser` config option. |
| `--reasoning-parser` | ✅ ACTIVE (text gen, configurable) | Required for reasoning models (DeepSeek-R1, QwQ etc.). |
| `--chat-template` | 🔜 CANDIDATE | Path to a Jinja2 chat template. Useful for models that don't ship a `tokenizer_config.json` template. |
| `--response-role` | ❌ NOT FOR US | Changes the `role` field in chat responses. We always pass-through as-is from vLLM to the ingress client. |

---

## 7. Security & Auth

| Flag | Status | Why it matters for us |
|---|---|---|
| `--api-key` | ✅ ACTIVE | See above. Ingress-to-worker auth. |
| `--ssl-certfile` / `--ssl-keyfile` | ❌ NOT FOR US | TLS termination is handled at the ingress layer (or service mesh). No need to terminate TLS at the vLLM worker. |
| `--allowed-origins` | ❌ NOT FOR US | CORS is irrelevant — vLLM worker is never browser-facing. |

---

## 8. Runner / Task Type

| Flag | Status | Why it matters for us |
|---|---|---|
| `--runner pooling` | ✅ ACTIVE (embedding, score) | Required for embedding/reranking models that use pooling runners instead of the generation runner. |
| `--convert embed` | ✅ ACTIVE (embedding, for converted models) | Auto-detects and converts generation-format embedding models. |

---

## Quick Reference: Currently Active Command

### Text Generation / Multimodal

```bash
python3 -m vllm.entrypoints.openai.api_server \
  --model <model-id> \
  --served-model-name <model-id> \
  --port 9000 \
  --trust-remote-code \
  --gpu-memory-utilization 0.80 \
  --dtype auto \
  --cpu-offload-gb 0 \
  --enable-prompt-tokens-details \
  --enable-per-request-metrics \
  --disable-log-stats \
  --uvicorn-log-level warning \
  [--max-model-len <N>] \
  [--tensor-parallel-size <N>] \
  [--enable-prefix-caching] \
  [--api-key <key>] \
  [--enable-auto-tool-choice --tool-call-parser <parser>] \
  [--reasoning-parser <parser>] \
  [--kv-cache-dtype fp8] \
  [--enforce-eager] \
  [--max-num-seqs <N>] \
  [--max-num-batched-tokens <N>]
```

### Embedding

```bash
python3 -m vllm.entrypoints.openai.api_server \
  --model <model-id> \
  --served-model-name <model-id> \
  --port 9000 \
  --trust-remote-code \
  --gpu-memory-utilization 0.80 \
  --dtype auto \
  --runner pooling \
  [--tensor-parallel-size <N>] \
  [--api-key <key>] \
  [--max-model-len <N>] \
  [--max-num-seqs <N>] \
  [--max-num-batched-tokens <N>] \
  [--convert embed] \
  [--enforce-eager]
```

### Score / Reranking

```bash
python3 -m vllm.entrypoints.openai.api_server \
  --model <model-id> \
  --served-model-name <model-id> \
  --port 9000 \
  --trust-remote-code \
  --gpu-memory-utilization 0.80 \
  --dtype auto \
  --runner pooling \
  [--tensor-parallel-size <N>] \
  [--api-key <key>] \
  [--max-model-len <N>] \
  [--max-num-seqs <N>] \
  [--max-num-batched-tokens <N>] \
  [--enforce-eager]
```

---

## Next Priority Additions

1. `--quantization` exposure — auto-detect from HF model config (`awq`, `gptq`, etc.) for cost optimization
2. `--scheduler-delay-factor` — micro-batching tuning once we have load data
3. OTel tracing (`--collect-detailed-traces` + `--otlp-traces-endpoint`) — needs tracing infra first

> **Ref:** https://docs.vllm.ai/en/stable/cli/serve/
