# Pipelines UI historical evidence

This dedicated Kubeflow-owned branch retains historical UI qualification evidence separately from application source and release tags. Do not merge it into master. It contains the original byte-identical archives and browsable snapshots; `index.json` records their SHA-256 digests and tested source identities.

The October 8 snapshot includes the per-file hash index for earlier large reports retained in the October 6 archive. Each result describes its recorded source, environment and limitations, not later application commits. These files are historical data; scripts or instructions inside snapshots are not automation for this branch.

Retain this branch for links from the modernization tracking issue and documentation. Publish links pinned to the evidence commit; future evidence should be appended without rewriting referenced commits. Routine CI artifacts do not need to be committed here.
