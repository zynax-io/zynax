# RFC-000: RFC Template

**RFC Number:** (assigned by maintainer)
**Title:** (short, descriptive)
**Author(s):** @github-handle
**Status:** Draft | Under Review | Accepted | Rejected | Withdrawn
**Created:** YYYY-MM-DD
**Last Updated:** YYYY-MM-DD
**Target Version:** (e.g., v0.3.0)

---

## Summary

One paragraph. What is being proposed and why does it matter?

---

## Motivation

What problem does this solve? What is the current pain point?
Include concrete examples if possible.

What happens if we do NOT do this?

---

## Detailed Design

The core of the RFC. Be specific. Include:

- API changes (proto diffs, REST endpoint changes)
- Data model changes (schema migrations if applicable)
- Behavioral changes (what changes from the user/agent perspective)
- Implementation approach
- Configuration changes

Code examples, proto snippets, diagrams are encouraged.

### Interface / API Sketch

```protobuf
// If proto changes are involved, show the diff here
```

```python
# If Python API changes are involved, show examples
```

---

## Alternatives Considered

What other approaches were evaluated? Why were they rejected?
This section prevents re-litigating decisions later.

---

## Impact Assessment

| Area | Impact | Notes |
|------|--------|-------|
| Breaking change? | Yes / No | Which services, which versions |
| Migration required? | Yes / No | Migration guide if yes |
| Performance impact | None / Minor / Major | Benchmarks if major |
| Security impact | None / Positive / Review needed | |
| Documentation needed | Yes / No | What needs updating |

---

## Open Questions

Unresolved items that need decision before this RFC can be accepted.

1. Question one?
2. Question two?

---

## Implementation Plan

If accepted, what are the implementation steps?

- [ ] Step 1
- [ ] Step 2
- [ ] Tests updated
- [ ] Docs updated
- [ ] ADR created

---

## References

- Related ADR: `docs/adr/ADR-XXX-*.md`
- Related issue: #XXX
- External references
