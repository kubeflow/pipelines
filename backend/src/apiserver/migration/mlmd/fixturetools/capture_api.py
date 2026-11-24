import json
import os
from pathlib import Path
from urllib.request import urlopen

output_dir = Path(
    os.environ.get('KFP218_OUTPUT')
    or Path(__file__).resolve().parents[1] / 'testdata' / 'release_218'
)
host = os.environ.get('KFP_HOST', 'http://127.0.0.1:18080').rstrip('/')
runs_path = output_dir / 'runs.json'
runs = json.loads(runs_path.read_text())
api_runs = {}
for entry in runs:
    with urlopen(
        f'{host}/apis/v2beta1/runs/{entry["run_id"]}', timeout=15
    ) as response:
        body = json.load(response)
    entry['status'] = body['state']
    api_runs[entry['case']] = body

runs_path.write_text(json.dumps(runs, indent=2) + '\n')
(output_dir / 'api_runs.json').write_text(json.dumps(api_runs, indent=2) + '\n')
