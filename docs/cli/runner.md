# CLI: Remote dispatch

Create durable remote work from the current Git repository and observe it through the Hub.

```bash
# Ad-hoc work; general scheduler matching
synchestra runner dispatch --prompt "Update dependencies" --model sonnet

# Committed SpecScore targets
synchestra runner dispatch --plan PLAN-42 --runner personal-vm
synchestra runner dispatch --task PLAN-42#task-2 --profile balanced

# Observe and control the returned durable ID
synchestra runner dispatch status dsp_01HXYZ
synchestra runner dispatch logs dsp_01HXYZ --cursor 0
synchestra runner dispatch retry dsp_01HXYZ --reason "transient failure"
synchestra runner dispatch cancel dsp_01HXYZ --reason "superseded"
```

Set `SYNCHESTRA_TOKEN` for Bearer authentication and optionally `SYNCHESTRA_URL` for a non-default Hub. `SYNCHESTRA_ACTOR` changes client provenance only; Hub authorization always derives from the Bearer identity.

Creation resolves the credential-free origin, full immutable `HEAD` revision, branch audit ref, project ID, and subdirectory without changing the checkout. SCP-style SSH origins such as `git@host:org/repo.git` remain SSH clone URLs so remote workers can use their authorized Git transport, while canonical repository IDs omit the transport user. No SSH command fallback runs in the CLI. Dirty/staged/untracked files remain untouched and are not included in the immutable remote snapshot. Ad-hoc prompts do not create Tasks.

Text status output includes the scheduler-resolved agent, model, effort, and routing reason. Completed attempts show the published branch, commit, and validation evidence; failed attempts show the failure stage, stable code, message, and log reference.

Use `--format json` for a single JSON object. Success contains `resolved` plus the operation payload; failure contains `resolved` plus `error`.

Exit codes specific to dispatch are `80` runner not found, `81` no eligible worker, `82` incompatible protocol, `101` unauthenticated, and `102` Hub unreachable. Standard CLI codes cover invalid arguments, conflicts, not-found, invalid state, and unexpected failures.

## Open Questions

None at this time.
