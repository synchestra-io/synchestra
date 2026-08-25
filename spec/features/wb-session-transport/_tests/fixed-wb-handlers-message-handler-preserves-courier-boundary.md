---
format: https://specscore.md/scenario-specification
---

# Scenario: Message handler preserves courier boundary

**Validates:** [wb-session-transport#req:fixed-wb-handlers](../README.md#req-fixed-wb-handlers)

## Steps

GIVEN an authorized typed message for a completed handoff
WHEN the fixed WB message handler is invoked
THEN Synchestra returns the opaque WB receipt without interpreting tmux or lineage

## Detected Surface

process

## TODO

- [ ] Pick Rehearse driver
- [ ] Wire up fixtures
- [ ] Implement assertion

---
*This document follows the https://specscore.md/scenario-specification*
