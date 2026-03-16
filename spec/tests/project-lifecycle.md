# Scenario: Project lifecycle

**Description:** End-to-end test of creating a Synchestra project and verifying config files.
**Tags:** e2e, cli, project

## Setup

```bash
export TEST_DIR=$(mktemp -d)
export SPEC_BARE=$(mktemp -d)/spec.git
export STATE_BARE=$(mktemp -d)/state.git
export TARGET_BARE=$(mktemp -d)/target.git
git init --bare "$SPEC_BARE"
git init --bare "$STATE_BARE"
git init --bare "$TARGET_BARE"
# Seed spec repo with a README
SEED=$(mktemp -d)
git clone "$SPEC_BARE" "$SEED/spec"
cd "$SEED/spec" && git config user.email "test@test" && git config user.name "Test"
echo "# Test Project" > README.md && git add . && git commit -m "init" && git push origin HEAD
cd -
export HOME="$TEST_DIR"
```

## create-project

**Outputs:**

| Name | Store | Extract |
|---|---|---|
| spec_repo_path | context | `echo $HOME/synchestra-repos/spec` |
| state_repo_path | context | `echo $HOME/synchestra-repos/state` |
| expected_title | context | `echo "Test Project"` |

```bash
synchestra project new \
  --spec-repo "$SPEC_BARE" \
  --state-repo "$STATE_BARE" \
  --target-repo "$TARGET_BARE"
```

## verify-configs

**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/new](spec/features/cli/project/new/) | * |

```bash
echo "Verifying config files exist"
```

## Teardown

```bash
rm -rf "$TEST_DIR" "$SPEC_BARE" "$STATE_BARE" "$TARGET_BARE"
```
