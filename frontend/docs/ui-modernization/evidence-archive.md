# Historical qualification evidence

Generated screenshots, runtime manifests, measurements and dated qualification records are retained on a [Kubeflow-owned evidence branch](https://github.com/kubeflow/pipelines/tree/codex/ui-modernization-evidence) rather than in application source. Routine CI runs upload Actions artifacts; evidence intended for long-term retention must be copied to project-owned durable storage before those artifacts expire.

- [October 8 archive](https://raw.githubusercontent.com/kubeflow/pipelines/d9aed93d6184be1f65d36066b40eb92028a2aa25/archives/kfp-ui-evidence-54f78798f.tar.gz): the complete documentation/evidence tree at `54f78798fea94aecc64006a8296a9473f453d55a`, including the 271 remaining generated files removed from the checkout.
- [October 6 archive](https://raw.githubusercontent.com/kubeflow/pipelines/d9aed93d6184be1f65d36066b40eb92028a2aa25/archives/kfp-ui-modernization-evidence-a058b109.tar.gz): the earlier complete tree, including large reports already removed before October 8.
- [Archive and file SHA-256 index](archived-evidence.json): current removed-file inventory plus the earlier archive inventory. The dedicated evidence commit retains browsable snapshots used by historical file links.

The archives and snapshots are pinned to evidence commit `d9aed93d6184be1f65d36066b40eb92028a2aa25`; their original SHA-256 hashes are unchanged. The branch is separate from master and must not be merged into application source. It is not a Kubeflow Pipelines product release. Each report applies only to its recorded source, assets, environment and limitations. Historical success does not qualify a later commit. Current implementation status belongs in [the tracking issue](https://github.com/kubeflow/pipelines/issues/14572) and the associated PR checks.

Verify the downloaded archive against its SHA-256 in the index before extracting into a separate directory. Then compare the archived files against the listed per-file hashes. Do not overlay historical captures on a checkout used for a new run or commit new dated CI output directories.
