# September 30 hosted qualification evidence

Large generated reports are preserved in the [evidence archive](../evidence-archive.md).
Original hash manifests apply to the extracted archive.

This is evidence for implementation head `20d87a3fd2e52eb6b33f1fe7beead25efbf4cc95`, tested merge `da6ec8bebac0614fcb1939a92aff4a7e1672b9b6`. It is not full release qualification: editor acceptance, one expanded browser lane and deployment/rollback verification remain open.

## Performance

[Run 36734476457](https://github.com/kubeflow/pipelines/actions/runs/36734476457) passed the accepted timing, payload and layout budgets. The independent audit verifies all 84 matched observations, 28 larger-workload observations and 84 compressed trace hashes, including 2,236,523 parsed trace events. Sources, fixture/build/harness hashes, individual observations, trace hashes and full-precision comparisons are retained in `performance.json`; `hosted-protocol.json` contains the effective instrumentation.

Seven interleaved pairs use fresh contexts on one runner, 4x CPU slowdown, 150 ms latency, 1.6 Mbit/s download and 0.75 Mbit/s upload. Browser process and OS caches remain warm. These fixture measurements do not establish deployed latency or field percentiles. Transfer timings use the fixture server's uncompressed assets; entry gzip is a separate payload measure.

| Measure | Legacy median | Candidate median |
| --- | ---: | ---: |
| Runs ready | 17,086 ms | 13,220 ms |
| Details ready | 16,924 ms | 13,307 ms |
| Compare ready | 17,120 ms | 13,384 ms |
| Filter | 978 ms | 775 ms |
| Open run | 1,048 ms | 966 ms |
| Open task | 206 ms | 95 ms |
| First editor display | 291 ms | 3,639 ms |

Entry JS/CSS gzip is 596,806 bytes versus 785,657 bytes, a 24.0% reduction. All 21 candidate initial-route samples have zero observed CLS. Filter CLS is 0.013583 in all seven trials; source rectangles identify only matching-row upward compaction and controls remain within one CSS pixel. Larger-workload medians remain within the accepted allowance against their separately pinned checkpoint; their unthrottled automation timings are not the same endpoint as the table above.

First editor display is not covered by the accepted timing gate. It compares complete read-only model content, fonts and two frames. Candidate YAML worker readiness is separately measured at a 4,439 ms median. The immutable legacy build lacks its worker asset: all seven actual 404s are retained, so there is no equivalent working legacy-worker timing. Editor acceptance remains pending.

## Functional coverage

[Engine run 36734475827](https://github.com/kubeflow/pipelines/actions/runs/36734475827) passed 432 distinct lane/case combinations across nine OS/engine lanes with no skips, plus 1,760 UI tests and 1,103 server tests. Every lane's 22 build-file hashes match the shared production build. Engine coverage does not establish actual vendor or device qualification.

[Vendor run 36734476076](https://github.com/kubeflow/pipelines/actions/runs/36734476076) has 16 independently passing lanes, 405 checks and 187 verified screenshot hashes. Both iPhone annual families and iPad 26 pass, including native Safari action evidence. The iPad 27 job timed out installing Appium before UI checks; this failed lane does not count as qualified and the workflow remains failed. Available simulators do not establish physical-device, native IME, pinch-zoom or full accessibility coverage. Provider-dependent policy versions remain separate follow-ups.

## Retention and reproducibility

`record-hashes.json` identifies the exact retained JSON records. The raw performance record includes each compressed trace's SHA256 and each build asset's source identity and SHA256. The vendor audit verifies native actions and identities; `vendor-records` retains original workflow/environment/result records and a file-hash inventory for screenshots and XML readbacks, including failed-attempt diagnostics. The engine audit includes the full shared build inventory.

Actions artifacts, including complete traces, screenshots, static builds and logs, expire after the repository's seven-day retention window. These selected sanitized records remain in Git; retaining their hashes does not make expired trace or screenshot bytes retrievable. No credentials, browser storage or private authentication snapshots are included.
