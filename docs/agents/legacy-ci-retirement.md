# Legacy v2 workflow retirement transition

Tracking: [#14950](https://github.com/kubeflow/pipelines/issues/14950).
Coverage and execution criteria are in the
[assertion ledger](../../backend/test/v2/api/legacy-migration.md).

This is a rollout plan, not an activated policy exception. The legacy workflow
remains required wherever its trusted-base trigger applies. The qualification
changes do not remove it from `ci-passed`.

## Establish retained evidence

Before proposing a policy activation, record two consecutive successful paired
comparisons on the same source revision across MySQL, MySQL with TLS,
PostgreSQL, and Kubernetes storage. Keep attempt-specific old/new reports,
source SHA, run URLs, timing, and all required scenario results on #14950.
Qualify both opt-in recurring-run service-account scenarios, including the
persisted authorization assertions. Missing, skipped, pending, or expired
reports do not establish the required execution evidence.

## Activate policy before deleting the workflow

`ci-passed` reads workflow requirements from immutable trusted-base YAML. A PR
that deletes the legacy workflow while its base still requires that workflow
cannot pass by presenting only a successful API workflow. Editing the head's
inventory, trigger, or policy does not authorize that removal.

After qualification, use two separately reviewed PRs:

1. The policy activation PR must retain and run the legacy workflow. Introduce
   an explicit retirement contract read only from the immutable trusted base,
   scoped to `legacy-v2-api-integration-tests.yml`, with
   `api-server-tests.yml` as its required replacement. Link the accepted
   qualification evidence. Validate the contract against base workflow data;
   fail closed if the replacement is absent or its applicable triggers no
   longer cover the retired lane. No head-provided flag, label, changed-file
   field, or inventory may activate it. Do not generalize the existing upgrade
   workflow pause into an arbitrary exclusion list.
2. After the activation PR merges, verify the live publisher uses that trusted
   base and that pending or failing replacement runs block `ci-passed`. Only
   then open the deletion PR. Remove the legacy workflow and unused helpers,
   regenerate the workflow inventory and completion selector, and preserve
   webhook callers and shared fixtures. Retain the replacement execution
   manifest and conditional service-account lane. Verify fresh replacement
   evidence on the deletion PR before merging.

The activation implementation must test both old and activated base revisions:
old base plus deletion head blocks; activation head cannot exempt itself; an
activated base accepts deletion only with applicable successful replacement
runs; absent, failed, stale, or wrong-head replacement evidence blocks. Add
loader tests proving the activation contract comes from the exact trusted-base
SHA, never the caller's checkout. These activation tests are still required;
the qualification PR does not implement or claim that transition.

Current verifier fixtures cover the old-base deletion refusal, both suites
being required before activation, and the final retired-base requirement for
successful replacement runs:

```sh
cd .github/resources/scripts
python3 -m unittest ci_expected_workflows_test ci_base_workflows_test
```

These are preservation tests of existing behavior, not a red-before/fixed-after
claim. The synthetic retired-base fixture is not evidence of deployed policy.

## Rollback and completion

If activation fails, leave the legacy workflow running and revert the contract;
do not bypass `ci-passed` or remove branch protection. If deletion exposes a
coverage gap, restore the legacy workflow and its required status behavior
before declaring parity restored. Keep source and reports available for that
rollback. Verify deployed merge protection and replacement enforcement after
each merge, then record final runner-time and wall-time measurements on #14950.
