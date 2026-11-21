# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

"""Container entry point for the two-profile upgrade pipeline fixture."""
import json
from pathlib import Path
import sys

label, score, dataset_path, executor_input = sys.argv[1:]
dataset = Path(dataset_path)
dataset.parent.mkdir(parents=True, exist_ok=True)
dataset.write_text(f"value\n{label}\n", encoding="utf-8")
preview = {
    "version": 1,
    "kind": "dataset",
    "format": "csv",
    "columns": [{"name": "value", "dtype": "string"}],
    "rows": [[label]],
    "total_rows": 1,
    "total_columns": 1,
    "sample": {
        "method": "head",
        "rows_truncated": False,
        "columns_truncated": False,
        "values_truncated": False,
        "redacted": False,
        "rows_omitted": False,
    },
}
Path(str(dataset) + ".kfp-preview.json").write_text(
    json.dumps(preview), encoding="utf-8")
output_file = Path(json.loads(executor_input)["outputs"]["outputFile"])
output_file.parent.mkdir(parents=True, exist_ok=True)
output_file.write_text(
    json.dumps({
        "artifacts": {
            "output_gcs": {"artifacts": [{
                "metadata": {"_kfp_preview": {"version": 2, "storage": "sibling"}}
            }]},
            "metrics": {"artifacts": [{
                "metadata": {"profile_upgrade_score": int(score)}
            }]},
        }
    }),
    encoding="utf-8",
)
