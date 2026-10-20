# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
"""Test-only broker: exchange a projected ServiceAccount JWT for real STS keys."""

import hashlib
import json
import os
import time
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from pathlib import Path


def refresh(endpoint, role, token_path, directory, generation):
    body = urllib.parse.urlencode(
        {
            'Action': 'AssumeRoleWithWebIdentity',
            'Version': '2011-06-15',
            'RoleArn': role,
            'RoleSessionName': 'artifact-broker-' + str(generation),
            'WebIdentityToken': token_path.read_text().strip(),
        }
    ).encode()
    request = urllib.request.Request(endpoint, data=body)
    with urllib.request.urlopen(request, timeout=5) as response:
        tree = ET.fromstring(response.read(65536))
    values = {
        element.tag.rsplit('}', 1)[-1]: element.text for element in tree.iter()
    }
    if not all(
        values.get(key)
        for key in (
            'AccessKeyId',
            'SecretAccessKey',
            'SessionToken',
            'Expiration',
        )
    ):
        raise RuntimeError('STS response lacks required credentials')
    temporary = directory / 'credentials.tmp'
    temporary.write_text(
        '[artifact-reader]\naws_access_key_id = '
        + values['AccessKeyId']
        + '\naws_secret_access_key = '
        + values['SecretAccessKey']
        + '\naws_session_token = '
        + values['SessionToken']
        + '\n'
    )
    temporary.chmod(0o600)
    temporary.replace(directory / 'credentials')
    status = {
        'generation': generation,
        'expiration': values['Expiration'],
        'fingerprint': hashlib.sha256(
            values['AccessKeyId'].encode()
        ).hexdigest(),
    }
    temporary = directory / 'status.tmp'
    temporary.write_text(json.dumps(status))
    temporary.replace(directory / 'status.json')


def main():
    directory = Path('/var/run/kfp-artifacts')
    token_path = Path('/var/run/secrets/artifact-storage/token')
    generation = 0
    while True:
        if not (directory / 'pause').exists():
            try:
                refresh(
                    os.environ['STS_ENDPOINT'],
                    os.environ['STORAGE_ROLE'],
                    token_path,
                    directory,
                    generation + 1,
                )
                generation += 1
            except Exception as error:
                # Neither JWTs nor STS responses may reach test logs.
                print(
                    'Credential refresh failed: ' + type(error).__name__,
                    flush=True,
                )
        time.sleep(4)


if __name__ == '__main__':
    main()
