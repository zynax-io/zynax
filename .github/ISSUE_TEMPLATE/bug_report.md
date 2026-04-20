---
name: Bug Report
about: Something is broken — behaviour differs from what the .feature file specifies
labels: ["type: bug", "status: needs-triage"]
assignees: ''
---

<!--
Before filing a bug:
  1. Search existing issues — this may already be reported.
  2. Check the relevant .feature file in services/<service>/tests/features/ to
     confirm the expected behaviour is defined there. If it is not, consider
     opening a Feature Request instead.
  3. Reproduce with the latest main or the latest release.
-->

## Which feature is broken?

Link to the `.feature` file and scenario that defines the expected behaviour:

```
services/<service>/tests/features/<feature>.feature
Scenario: <scenario name>
```

---

## What actually happens?

Describe the actual (broken) behaviour. Be specific — what output, error, or
state did you observe?

---

## What should happen instead?

Describe the expected behaviour (as defined in the feature file or, if not yet
defined, as you understand it should work).

---

## Steps to Reproduce

Minimum steps to trigger the bug:

1.
2.
3.

If you have a minimal reproducer (a test, a YAML manifest, a sequence of commands),
paste it here:

```bash
# Commands or YAML
```

---

## Environment

| Field | Value |
|-------|-------|
| Keel version / commit | |
| Service affected | |
| OS / Arch | |
| Docker version | |
| Go version (if building locally) | |
| Python version (if building locally) | |

---

## Logs / Error Output

Paste relevant structured log output or error messages. Remove any sensitive data.

```
```

---

## Impact

- [ ] Data loss or corruption
- [ ] Security-relevant (stop — use the [Security Advisory](https://github.com/keel-io/keel/security/advisories/new) form instead)
- [ ] Workflow execution blocked
- [ ] Degraded behaviour (workaround exists)
- [ ] Cosmetic / low impact

---

## Additional Context

Any other context: related issues, recent changes, ADRs, suspected root cause.
