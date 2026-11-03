"""Summarize complete image scans for GitHub without changing raw evidence."""

import argparse
import hashlib
import json
import math
from pathlib import Path
from urllib.parse import quote, urlparse

from .osv_image_scan import SCANNER, validate_sarif


def _score(properties: dict) -> float | None:
    value = properties.get('security-severity')
    if isinstance(value, bool):
        return None
    try:
        number = float(value)
    except (ValueError, TypeError):
        return None
    return number if math.isfinite(number) and 0 <= number <= 10 else None


def _validate_invocations(run: dict) -> None:
    invocations = run.get('invocations', [])
    if not isinstance(invocations, list):
        raise ValueError('SARIF invocations must be a list')
    for invocation in invocations:
        if not isinstance(invocation, dict):
            raise ValueError('Invalid SARIF invocation')
        if invocation.get('executionSuccessful') is False:
            raise ValueError('Cannot summarize an unsuccessful scan')
        for key in (
            'toolExecutionNotifications',
            'toolConfigurationNotifications',
        ):
            notifications = invocation.get(key, [])
            if not isinstance(notifications, list):
                raise ValueError('Invalid SARIF notifications')
            if any(
                not isinstance(item, dict) or item.get('level') == 'error'
                for item in notifications
            ):
                raise ValueError('Cannot summarize scan errors')


def summarize(
    sarifs: list[dict],
    *,
    identity: str,
    image: str,
    evidence_url: str,
    platforms: list[str],
) -> dict:
    """Return one stable image result, or none after a complete clean scan.

    Callers must verify scan completeness and immutable image bindings
    before calling this presentation-only helper. Identity describes a
    logical image, independent of tags, digests, architectures and the
    current advisory set.
    """
    if not sarifs or len(sarifs) != len(platforms):
        raise ValueError('Every image scan must have a platform')
    if any(not isinstance(p, str) or not p.strip() for p in platforms):
        raise ValueError('Image platforms must be nonempty strings')
    if len(set(platforms)) != len(platforms):
        raise ValueError('Image platforms must be unique')
    if not isinstance(identity, str) or not identity.strip():
        raise ValueError('A stable logical image identity is required')
    if not isinstance(image, str) or not image.strip():
        raise ValueError('An image reference is required')
    parsed = urlparse(evidence_url)
    if parsed.scheme != 'https' or not parsed.netloc:
        raise ValueError('Raw evidence requires an HTTPS URL')

    ids: set[str] = set()
    occurrences = 0
    unknown = 0
    scores: list[float] = []
    for sarif in sarifs:
        results = validate_sarif(sarif)
        run = sarif['runs'][0]
        if 'results' not in run:
            raise ValueError('Image scan must explicitly report its results')
        _validate_invocations(run)
        rules = {r['id']: r for r in run['tool']['driver'].get('rules', [])}
        occurrences += len(results)
        for result in results:
            ids.add(result['ruleId'])
            properties = rules[result['ruleId']].get('properties', {})
            result_properties = result.get('properties', {})
            reported_scores = [
                score
                for value in (properties, result_properties)
                if isinstance(value, dict)
                and (score := _score(value)) is not None
            ]
            if reported_scores:
                scores.extend(reported_scores)
            else:
                unknown += 1

    fingerprint = hashlib.sha256(identity.encode()).hexdigest()
    rule_id = f'image-vulnerabilities/{fingerprint}'
    severity = (
        f'Highest reported security score: {max(scores):g}/10.'
        if scores
        else 'No numeric security scores were reported.'
    )
    text = (
        f'Image: {image}\n'
        f'Logical image: {identity}\n'
        f'Architectures: {", ".join(sorted(platforms))}\n'
        f'{len(ids)} distinct advisory IDs; {occurrences} raw result occurrences '
        '(including repeated packages and architectures).\n'
        f'{severity} {unknown} occurrences have unknown security scores; '
        'unknown does not mean low risk.\n'
        f'Full raw JSON/SARIF, package versions and scan provenance: {evidence_url}\n'
        'This is a reporting summary. Release security gates evaluate the full '
        'findings; dismissing this summary does not waive them.'
    )
    evidence_link = quote(evidence_url, safe=':/?=&%#@+;,')
    markdown = (
        f'**{len(ids)} distinct advisory IDs** across **{occurrences} raw '
        'result occurrences** (including repeated packages and architectures).\n\n'
        f'{severity} **{unknown} occurrences have unknown security scores**; '
        'unknown does not mean low risk.\n\n'
        f'[View full raw JSON/SARIF, package versions and scan provenance]'
        f'(<{evidence_link}>).\n\n'
        'This is a reporting summary. Release security gates evaluate the full '
        'findings; dismissing this summary does not waive them.'
    )
    properties: dict = {'tags': ['security', 'container-image']}
    if scores:
        properties['security-severity'] = f'{max(scores):g}'
    rule = {
        'id': rule_id,
        'name': 'ImageVulnerabilities',
        'shortDescription': {'text': f'Vulnerabilities in {image}'[:1024]},
        'fullDescription': {
            'text': 'One summary per logical image. See the linked raw evidence '
            'for all advisory and package details.'
        },
        'help': {'text': text, 'markdown': markdown},
        'properties': properties,
    }
    results = []
    if occurrences:
        results.append(
            {
                'ruleId': rule_id,
                'ruleIndex': 0,
                'level': 'warning',
                'message': {'text': text},
                'partialFingerprints': {'primaryLocationLineHash': fingerprint},
                'locations': [
                    {
                        'physicalLocation': {
                            'artifactLocation': {
                                'uri': '.github/workflows/osv-scanner.yml'
                            },
                            'region': {'startLine': 1},
                        }
                    }
                ],
                'properties': {
                    'logicalImage': identity,
                    'image': image,
                    'platforms': sorted(platforms),
                    'advisoryCount': len(ids),
                    'rawResultCount': occurrences,
                    'unknownSeverityCount': unknown,
                    'evidenceUrl': evidence_url,
                },
            }
        )
    return {
        '$schema': 'https://json.schemastore.org/sarif-2.1.0.json',
        'version': '2.1.0',
        'runs': [
            {
                'tool': {'driver': {**SCANNER, 'rules': [rule]}},
                'results': results,
            }
        ],
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--input', type=Path, action='append', required=True)
    parser.add_argument('--identity', required=True)
    parser.add_argument('--image', required=True)
    parser.add_argument('--evidence-url', required=True)
    parser.add_argument('--platform', action='append', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    report = summarize(
        [json.loads(path.read_text()) for path in args.input],
        identity=args.identity,
        image=args.image,
        evidence_url=args.evidence_url,
        platforms=args.platform,
    )
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + '\n')


if __name__ == '__main__':
    main()
