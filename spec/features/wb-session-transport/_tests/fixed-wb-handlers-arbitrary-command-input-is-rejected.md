---
format: https://specscore.md/scenario-specification
---

# Scenario: Arbitrary command input is rejected

**Validates:** [wb-session-transport#req:fixed-wb-handlers](../README.md#req-fixed-wb-handlers)

## Steps

GIVEN an unknown handler or payload containing command syntax
WHEN the runner validates the invocation
THEN execution is refused and request data is never treated as a command or logged as payload content

## Detected Surface

process

## TODO

- [ ] Pick Rehearse driver
- [ ] Wire up fixtures
- [ ] Implement assertion

---
*This document follows the https://specscore.md/scenario-specification*
