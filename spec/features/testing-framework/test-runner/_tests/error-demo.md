# Scenario: Error reporting demo

**Description:** Demonstrates how failures and errors are reported in real-time.
**Tags:** demo, manual

## passing-step

```bash
echo "this step succeeds"
```

## failing-step

```bash
echo "about to fail..." >&2
exit 1
```

## after-failure

```bash
echo "this still runs"
```
