#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import collections
import json
from pathlib import Path
import sys


def summarize(path):
    invocations = collections.Counter()
    responses = collections.Counter()
    quotas = {}
    for line in Path(path).read_text().splitlines():
        fields = line.split("\t")
        if fields[0] == "I" and len(fields) == 3:
            invocations[(fields[1], int(fields[2]))] += 1
        elif fields[0] == "R" and len(fields) == 8:
            responses[(fields[1], int(fields[2]))] += 1
            snapshot = {
                key: int(value)
                for key, value in zip(("limit", "remaining", "used",
                                       "reset"), fields[3:7])
                if value != "-"
            }
            if snapshot:
                resource = quotas.setdefault(
                    fields[7],
                    {
                        "observations": 0,
                        "first": snapshot,
                    },
                )
                resource["observations"] += 1
                resource["last"] = snapshot
    return {
        "event": "github_request_accounting",
        "version": 1,
        "scope": "artifact_readiness",
        "measurement": "gh_response_headers",
        "coverage": "Observed gh API response headers including pagination; "
                    "opaque transport retries and redirects are excluded.",
        "invocations": sum(invocations.values()),
        "responses": sum(responses.values()),
        "counts": [{
            "category": category,
            "status": status,
            "count": count
        } for (category, status), count in sorted(responses.items())],
        "exits": [{
            "category": category,
            "exit_code": code,
            "count": count
        } for (category, code), count in sorted(invocations.items())],
        "rate_limits": quotas,
    }


if __name__ == "__main__":
    print(json.dumps(summarize(sys.argv[1]), separators=(",", ":")))
