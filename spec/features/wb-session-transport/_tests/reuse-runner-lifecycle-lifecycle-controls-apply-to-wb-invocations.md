---
format: https://specscore.md/scenario-specification
---

# Scenario: Lifecycle controls apply to WB invocations

**Validates:** [wb-session-transport#req:reuse-runner-lifecycle](../README.md#req-reuse-runner-lifecycle)

## Steps

GIVEN a queued or running WB invocation
WHEN existing runner status, logs, or cancellation operations address it
THEN they expose or change the durable fenced attempt lifecycle without exposing payload bytes

## Detected Surface

data

## TODO

- [ ] Pick Rehearse driver
- [ ] Wire up fixtures
- [ ] Implement assertion

---
*This document follows the https://specscore.md/scenario-specification*
