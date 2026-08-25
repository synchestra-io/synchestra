---
format: https://specscore.md/scenario-specification
---

# Scenario: Retry returns one WB receipt

**Validates:** [wb-session-transport#req:idempotent-handoff-delivery](../README.md#req-idempotent-handoff-delivery)

## Steps

GIVEN a completed WB invocation whose response was lost
WHEN the same handoff ID and digest are submitted again
THEN the stored receipt returns without a second lease or handler launch

## Detected Surface

data

## TODO

- [ ] Pick Rehearse driver
- [ ] Wire up fixtures
- [ ] Implement assertion

---
*This document follows the https://specscore.md/scenario-specification*
