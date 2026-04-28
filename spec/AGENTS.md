# spec/ — Engineering Contract

> The `spec/` directory is **Layer 1 — Intent** of the three-layer model.
> Everything here is YAML. No Go or Python. YAML in `spec/` is never imported
> by code — it is compiled by the Workflow Compiler.

---

## What Lives Here

```
spec/
├── schemas/                      ← JSON Schema for all manifest kinds
│   ├── workflow.schema.json
│   ├── agent-def.schema.json
│   ├── policy.schema.json
│   └── capability.schema.json
├── asyncapi/
│   └── zynax-events.yaml         ← All 11 async event channels
└── workflows/examples/           ← Reference YAML manifests
    ├── code-review.yaml
    ├── ci-pipeline.yaml
    └── research-task.yaml
```

---

## YAML Authoring Rules

1. Always include `kind` and `apiVersion` — required for schema validation.
2. Always include `metadata.name` and `metadata.namespace`.
3. Capabilities are `snake_case` — `request_review`, not `RequestReview`.
4. **Never reference agent names** — only capability names. Routing is the platform's job.
5. State names are lowercase, descriptive — `review`, `fix`, not `s1`, `s2`.
6. Use `{{ .event.field }}` for input templates (Jinja2-style, evaluated by compiler).
7. At least one state must have `type: terminal`.
8. Validate locally before applying: `make validate-spec`.

---

## AsyncAPI Event Versioning

### Non-breaking changes (safe)
Adding optional fields to the CloudEvent `data` payload is non-breaking.
Consumers MUST ignore unknown fields.

### Breaking changes (field removal, rename, type change)
1. Create a new channel with `.v2` suffix: `zynax.workflow.started.v2`
2. Mark the old channel deprecated in the AsyncAPI spec (`x-zynax-deprecated: true`)
3. Publish to both channels for one full milestone (deprecation period)
4. Remove the old channel at the start of the next milestone

Every CloudEvent MUST include the `zynaxschemarev` extension:
`zynaxschemarev: "workflow/started@v1"` — increment `N` on every breaking change.

---

## Validation Commands

```bash
make validate-spec             # validate all specs (AsyncAPI + all schemas)
make validate-asyncapi         # validate AsyncAPI spec only
make validate-workflow-schema  # validate Workflow manifests only
make dry-run FILE=spec/workflows/examples/code-review.yaml
```

CI blocks merge if any `.yaml` in `spec/` fails schema validation.
