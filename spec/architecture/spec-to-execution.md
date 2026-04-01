# Spec-to-Execution Pipeline

How product intent becomes running work in Synchestra.

Synchestra's architecture separates three concerns — **what** to build, **how** to build it, and **who is building it right now** — across distinct artifacts and repositories. This document shows how they connect.

## The three layers

```mermaid
graph TB
    subgraph "Spec repo (intent)"
        F["Feature<br/>what to build"]
        P["Proposal<br/>what to change"]
        DP["Plan<br/>how to build it"]
    end

    subgraph "State repo (execution)"
        T["Tasks<br/>who is doing what"]
        A["Artifacts<br/>inter-task data"]
        TSB["Task Status Board<br/>visibility + claiming"]
    end

    subgraph "Code repo (output)"
        C["Branches<br/>implementation"]
    end

    F -->|new feature| DP
    P -->|change request| DP
    P -.->|incorporated| F
    DP -->|generates| T
    T --- TSB
    T -->|produces| A
    A -->|consumed by| T
    T -->|works on| C
```

Each layer has a different **mutability profile**:

| Layer            | Artifact          | Mutability          | Repository |
|------------------|-------------------|---------------------|------------|
| Intent (what)    | Feature spec      | Versioned, evolving | Spec       |
| Intent (what)    | Proposal          | Versioned until incorporated | Spec |
| Approach (how)   | Plan              | Mutable; snapshots track history | Spec   |
| Execution (who)  | Tasks             | Highly fluid        | State      |
| Execution (who)  | Artifacts         | Write-once per task | State      |
| Output           | Code branches     | Standard git flow   | Code       |

## End-to-end lifecycle

A complete cycle from idea to retrospective:

```mermaid
sequenceDiagram
    participant Human
    participant Feature as Feature Spec
    participant Proposal as Proposal
    participant Plan as Plan
    participant Tasks as Tasks
    participant Board as Status Board
    participant Code as Code Repo

    Human->>Feature: Define new feature
    Note over Feature: Status: Conceptual

    alt Change to existing feature
        Human->>Proposal: Submit change request
        Note over Proposal: draft → submitted → approved
        Proposal-->>Feature: Incorporated when implemented
    end

    Human->>Plan: Author plan
    Note over Plan: draft → in_review → approved

    Plan->>Tasks: Generate tasks from plan tasks
    Note over Tasks: planning → queued

    loop For each claimable task
        Tasks->>Board: Agent claims task
        Note over Board: queued → in_progress
        Tasks->>Code: Agent works on branch
        Tasks->>Tasks: Produce artifacts
        Tasks->>Board: Agent completes task
        Note over Board: in_progress → complete
    end

    Human->>Plan: Request deviation report
    Note over Plan: Compare plan vs actual
```

## Artifact relationships

Each artifact type references its neighbors, creating a navigable chain:

```mermaid
graph LR
    F["Feature"]
    PR["Proposal"]
    DP["Plan"]
    T["Task"]
    A["Artifact"]

    F -->|"Plans section"| DP
    PR -->|"Plan field"| DP
    DP -->|"Features header"| F
    DP -->|"Source field"| PR
    DP -->|"Task mapping"| T
    T -->|"Plan + Plan task"| DP
    T -->|"Inputs field"| A
    T -->|"Produces"| A
```

Every link is bidirectional or navigable in both directions. You can start at any node and trace the full chain:

- **From a feature:** see its plans, which show tasks, which show artifacts and code branches
- **From a task:** see its plan task, which shows the plan, which shows the feature and acceptance criteria
- **From a plan:** see both the original intent (feature/proposal) and the execution state (tasks + derived status)

## Status and mutability

The three core artifacts have fundamentally different mutability rules, and this is by design:

```mermaid
graph LR
    subgraph "Evolving"
        F["Feature spec<br/>──────────<br/>Living document.<br/>Updated when proposals<br/>are incorporated."]
    end

    subgraph "Snapshot-tracked"
        DP["Plan<br/>──────────<br/>Mutable document.<br/>Snapshots capture<br/>reference points.<br/>Never tracks status."]
    end

    subgraph "Fluid"
        T["Tasks<br/>──────────<br/>Added, split, cancelled,<br/>restructured during<br/>execution. This is normal."]
    end

    F --> DP
    DP --> T
```

**Why this matters:**
- Features **evolve** because the product definition changes over time. Proposals are the mechanism for controlled evolution.
- Plans are **mutable** — they evolve as execution reveals complexity. Snapshots (git hash + action + comment) capture reference points for review and retrospective.
- Tasks are **fluid** because real execution always deviates from the plan. Agents discover complexity, humans reprioritize, parallel work gets restructured. Freezing tasks would fight reality.

The plan bridges these two worlds. Snapshots mark the moments where intent crystallizes — approval, checkpoints, completion — while the plan itself remains a living document.

## Derived status: no duplication

Plans do not track task status. Instead, Synchestra derives a progress view on the fly by mapping plan tasks to their linked execution tasks:

```mermaid
graph LR
    subgraph "Plan (in spec repo)"
        S1["Task 1"]
        S2["Task 2"]
        S3["Task 3"]
    end

    subgraph "Tasks (live, in state store)"
        T1["Task 1 ✅"]
        T2["Task 2 🔵"]
        T2a["Task 2a ✅"]
        T2b["Task 2b ⏳"]
        T3["Task 3 ⏳"]
        TX["Task X 🔵<br/>(unplanned)"]
    end

    S1 -.->|task mapping| T1
    S2 -.->|task mapping| T2
    T2 --- T2a
    T2 --- T2b
    S3 -.->|task mapping| T3
```

The derived view shows:
- **Task 1:** complete (Task 1 is done)
- **Task 2:** in progress (Task 2 has sub-tasks, one done, one queued)
- **Task 3:** queued (Task 3 hasn't started)
- **Unplanned:** Task X exists but wasn't in the original plan

One source of truth (tasks), two views (flat plan progress for humans, deep task tree for agents).

## Repository boundaries

```mermaid
graph TB
    subgraph "Spec repo"
        direction TB
        SF["spec/features/"]
        SP["spec/plans/"]
    end

    subgraph "State repo"
        direction TB
        ST["tasks/"]
        SA["tasks/*/artifacts/"]
    end

    subgraph "Code repo(s)"
        direction TB
        CB["synchestra/* branches"]
    end

    SF -->|"plan references<br/>features"| SP
    SP -->|"tasks generated<br/>from plan tasks"| ST
    ST -->|"agents work on<br/>code branches"| CB
    ST --- SA
```

The spec repo holds everything about **intent and approach** (features, proposals, plans). The state repo holds everything about **execution** (tasks, artifacts, status boards). Code repos hold the **output** (implementation on branches). This separation ensures that high-frequency machine commits (task claims, status transitions) never pollute the spec or code history.

### Embedded state: single-repo alternative

For single-repo projects, all three layers can live in one repository using [embedded state](../features/embedded-state/README.md). State lives on an orphan branch (`synchestra-state`) checked out as a worktree at `.synchestra/`. The orphan branch has no shared history with `main`, preserving the same commit isolation as a dedicated state repo.

```mermaid
graph TB
    subgraph "One repo, two branches"
        direction TB
        subgraph "main branch (spec + code)"
            SF2["spec/features/"]
            SP2["spec/plans/"]
            CB2["source code"]
        end

        subgraph "synchestra-state branch (orphan → .synchestra/)"
            ST2["tasks/"]
            SA2["tasks/*/artifacts/"]
        end
    end

    SF2 -->|"plan references<br/>features"| SP2
    SP2 -->|"tasks generated<br/>from plan tasks"| ST2
    ST2 --- SA2
```

Set up with `synchestra project init`. See [Embedded State](../features/embedded-state/README.md) for details.

## Outstanding Questions

None at this time.
