---
format: https://specscore.md/scenario-specification
---

# Scenario: Typed WB handoff reaches fixed handler

**Validates:** [wb-session-transport#req:typed-runner-invocation](../README.md#req-typed-runner-invocation)

## Steps

GIVEN an authorized caller, eligible runner, and valid WB handoff payload
WHEN the registered WB receive handler is invoked
THEN only its configured argv runs and its opaque structured receipt is persisted

## Detected Surface

cli

## TODO

- [ ] Pick Rehearse driver
- [ ] Wire up fixtures
- [ ] Implement assertion

---
*This document follows the https://specscore.md/scenario-specification*
