# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Sanitized migration findings from stored workload evidence.

Stored JSON is not an original upload, a compiled workflow, or a record
of requests that will be made after upgrade. Keep these evidence
boundaries explicit.
"""

from collections import Counter
import ipaddress
import json
from urllib.parse import urlsplit

from kfp_inventory import field
from kfp_inventory import v2_template
from workload_inventory import RESOURCES

KINDS = dict(
    experiments='experiment',
    pipelines='pipeline',
    pipeline_versions='pipeline_version',
    runs='run',
    recurring_runs='recurring_run')
SIZE_FIELDS = ('upload_bytes', 'spec_bytes', 'update_body_bytes',
               'parameter_bytes', 'metrics_bytes')
MAX_FINDINGS = 10000


def finding(rule, status, resource, evidence, action, verification):
    return dict(
        rule=rule,
        status=status,
        resource=resource,
        evidence=evidence,
        action=action,
        verification=verification)


def validate_target(bundle):
    if bundle is None:
        return {}
    if bundle.get('runtime') not in (None, 'legacy-and-v2', 'v2-only'):
        raise ValueError('Unsupported target runtime.')
    limits = bundle.get('limits', {})
    if not isinstance(limits, dict) or set(limits) - set(SIZE_FIELDS):
        raise ValueError('Unsupported size limit.')
    for key, value in limits.items():
        if type(value) is not int or value <= 0:
            raise ValueError('Size limits must be positive byte counts.')
        if key in SIZE_FIELDS[:3] and value > 134217728:
            raise ValueError('Pipeline size limits cannot exceed 128 MiB.')
    if not isinstance(bundle.get('http_base_url', ''), str):
        raise ValueError('HTTP base must be a string.')
    evidence = bundle.get('resource_evidence', [])
    if not isinstance(evidence, list) or len(evidence) > 10000:
        raise ValueError('Resource evidence must be a bounded list.')
    seen = set()
    for row in evidence:
        if (not isinstance(row, dict) or
                row.get('resource_kind') not in KINDS.values() or
                not isinstance(row.get('resource_id'), str) or
                not row['resource_id']):
            raise ValueError('Resource evidence requires a kind and ID.')
        key = row['resource_kind'], row['resource_id']
        if key in seen:
            raise ValueError('Duplicate resource evidence.')
        seen.add(key)
        for name in SIZE_FIELDS:
            if name in row and (type(row[name]) is not int or row[name] < 0):
                raise ValueError(
                    'Measured sizes must be nonnegative byte counts.')
    return bundle


def template(record):
    spec = field(record, 'pipeline_spec', 'pipelineSpec')
    if spec is None:
        spec = record.get('_readiness_pipeline_spec')
    if isinstance(spec, str):
        try:
            spec = json.loads(spec)
        except ValueError:
            return 'unresolved', None
    if v2_template(spec):
        return 'v2_ir', spec
    if (isinstance(spec, dict) and
            spec.get('kind') in ('Workflow', 'WorkflowTemplate') and
            str(spec.get('apiVersion', '')).startswith('argoproj.io/') and
            isinstance(spec.get('spec'), dict)):
        return 'legacy_argo', spec
    return 'unresolved', spec if isinstance(spec, dict) else None


def _http_parts(value):
    if not isinstance(value, str) or any(
            ord(c) < 32 for c in value) or '\\' in value:
        return None
    try:
        parts = urlsplit(value)
        if (parts.scheme not in ('http', 'https') or not parts.hostname or
                parts.username is not None or parts.password is not None):
            return None
        port = parts.port if parts.port is not None else (
            443 if parts.scheme == 'https' else 80)
        host = parts.hostname.lower()
        # WHATWG URL normalizes encoded/Unicode hosts and unusual IPv4 forms
        # differently from urllib. Leave these unknown instead of false denials.
        if '%' in host or not host.isascii():
            return None
        if ':' in host:
            host = str(ipaddress.IPv6Address(host))
        elif host.rsplit('.', 1)[-1].isdigit() or host.startswith('0x'):
            host = str(ipaddress.IPv4Address(host))
        origin = parts.scheme + '://' + ('[' + host +
                                         ']' if ':' in host else host)
        if port != (443 if parts.scheme == 'https' else 80):
            origin += ':' + str(port)
        return parts, origin
    except ValueError:
        return None


def _artifact_uris(record, spec):
    # Inspect defined URI locations only, never arbitrary parameters or strings.
    values = []
    config = field(record, 'runtime_config', 'runtimeConfig')
    if isinstance(config, dict):
        values.append(field(config, 'pipeline_root', 'pipelineRoot'))
    if isinstance(spec, dict):
        values.append(
            field(spec, 'default_pipeline_root', 'defaultPipelineRoot'))
    # Source 2.17 task ArtifactList contains MLMD IDs, not artifact URIs.
    # Resolving them needs separately authorized metadata-store evidence.
    return [value for value in values if isinstance(value, str) and value]


def _artifact_findings(record, spec, resource, target):
    result, seen = [], set()
    for uri in _artifact_uris(record, spec):
        parsed = _http_parts(uri)
        if parsed:
            parts, origin = parsed
            base = _http_parts(target.get('http_base_url', '').strip())
            if base and (base[0].query or base[0].fragment):
                base = None
            status = 'unknown'
            evidence = 'Observed HTTP artifact origin ' + origin + '; paths and query values withheld. '
            if 'http_base_url' in target and not target['http_base_url'].strip(
            ):
                status = 'policy_rejection'
                evidence += 'The supplied target has no HTTP_BASE_URL, which rejects HTTP artifact reads.'
            elif base:
                base_parts, base_origin = base
                # Encoded/dot segments depend on the production URL normalizer.
                # Never claim a path match when reproducing its semantics is uncertain.
                ambiguous = any(
                    '%' in p or any(s in ('.', '..')
                                    for s in p.split('/'))
                    for p in (parts.path, base_parts.path))
                prefix = base_parts.path.removesuffix('/') + '/'
                if origin != base_origin or (
                        not ambiguous and not (parts.path == prefix[:-1] or
                                               parts.path.startswith(prefix))):
                    status = 'policy_rejection'
                    evidence += 'Origin/path is outside the supplied absolute HTTP_BASE_URL.'
                else:
                    evidence += 'Origin/path needs a real target artifact request; redirects, credentials, profile proxies and encoded paths are not inferred.'
            else:
                evidence += 'Target HTTP base is absent, invalid, or uses gateway mapping; destination behavior is unresolved.'
        else:
            provider = uri.split(':', 1)[0].lower()
            if provider not in ('s3', 'minio', 'gs', 'http', 'https'):
                provider = 'unresolved'
            status = 'unknown'
            evidence = 'Observed artifact provider ' + provider + '; location withheld. '
            evidence += 'Provider URI does not establish the effective storage endpoint, profile proxy, credentials or archive access.'
        key = status, evidence
        if key in seen:
            continue
        seen.add(key)
        result.append(
            finding(
                'workload.artifacts', status, resource, evidence,
                'Configure the approved target HTTP base or exact storage endpoints and namespace proxy; do not add unrestricted fetching.',
                'Read a representative artifact and archived log through the target UI with intended credentials; verify denied destinations too.'
            ))
    result.append(
        finding(
            'workload.artifactCoverage', 'unknown', resource,
            'Source 2.17 task artifacts are MLMD IDs; their URIs, runtime-selected locations and legacy workflow archives are not resolved by KFP API inventory.',
            'Sample task output artifacts, profile proxy settings and archived logs using authorized metadata-store evidence.',
            'Verify actual artifact reads after upgrade; a configured root or absent URI is not a pass.'
        ))
    return result


def assess(inventory, target=None):
    target = validate_target(target)
    findings, summary = [], Counter()
    evidence_rows = {
        (row['resource_kind'], row['resource_id']): row
        for row in target.get('resource_evidence', [])
    }
    pipelines = {
        field(p, 'pipeline_id', 'pipelineId'): p for p in inventory['pipelines']
    }
    for category, records in inventory.items():
        if category not in KINDS:
            continue
        for record in records:
            if len(findings) >= MAX_FINDINGS:
                return _bounded_result(findings, summary, truncated=True)
            record_id = field(record, *RESOURCES[category][:2])
            kind = KINDS[category]
            resource = kind + '/' + (record.get('namespace') or
                                     '_shared') + '/' + record_id
            summary[category] += 1
            for reason in record.get('_readiness_collection_errors', []):
                findings.append(
                    finding(
                        'workload.collection', 'unknown', resource, reason,
                        'Provide the missing resource and its verified parent namespace.',
                        'Rerun the scoped collection; a partial reference is never a successful check.'
                    ))
            ownership = record.get('_readiness_namespace_evidence',
                                   'unresolved')
            findings.append(
                finding(
                    'workload.ownership', 'unknown', resource,
                    'Namespace evidence: ' + ownership +
                    '. Namespace membership does not identify an initiating user or prove historical ownership.',
                    'Supply the intended initiating caller or scheduled-workflow controller in target-policy identities.',
                    'Verify target namespace authorization with that identity; do not infer it from display names.'
                ))
            if category in ('experiments', 'pipelines'):
                continue
            shape, spec = template(record)
            summary[shape] += 1
            status = 'unknown'
            message = 'Stored template classification: ' + shape + '. '
            if shape == 'legacy_argo' and target.get('runtime') == 'v2-only':
                status = 'policy_rejection'
                message += 'The supplied v2-only target cannot execute this legacy template.'
            elif shape == 'legacy_argo':
                message += 'Legacy compatibility depends on the selected release runtime; 2.18 release and post-MLMD master differ.'
            elif shape == 'v2_ir':
                message += 'V2 shape is observed; SDK/compiler, platform extensions and plugin mutations still require validation.'
            else:
                message += 'Stored template was absent, inaccessible or not recognized; no compatibility conclusion is possible.'
            findings.append(
                finding(
                    'workload.template', status, resource, message,
                    'Recompile unsupported legacy pipelines to V2 IR and recreate affected schedules; retain original sources.',
                    'Compile with the intended SDK/plugins and execute representative tasks on the exact target revision.'
                ))
            reference = record.get('_readiness_version_reference')
            if reference:
                resolved = reference.get('resolution') == 'observed'
                parent = pipelines.get(reference.get('pipeline_id'))
                cross_namespace = (
                    resolved and parent and parent.get('namespace')
                    not in ('', '-', record.get('namespace')))
                rejection = cross_namespace and target.get(
                    'multi_user') is True and target.get('shared_read') is False
                findings.append(
                    finding(
                        'workload.reference',
                        'policy_rejection' if rejection else 'unknown',
                        resource, 'Reference kind: ' + reference['kind'] +
                        '; resolution: ' + reference['resolution'] + '. ' +
                        ('Private pipeline belongs to another namespace and supplied target shared-read is disabled.'
                         if rejection else
                         'A moving latest reference is only a snapshot; resolved references still need target read permission.'
                        ),
                        'Pin a retained pipeline version in the run namespace or intentionally publish it as shared; update consumers.',
                        'Verify the initiating user and controller can read the referenced target pipeline.'
                    ))
            findings.extend(_artifact_findings(record, spec, resource, target))
            supplied = evidence_rows.get((kind, record_id), {})
            observed = []
            for name in SIZE_FIELDS:
                if name not in supplied:
                    continue
                value, limit = supplied[name], target.get('limits',
                                                          {}).get(name)
                observed.append(name)
                findings.append(
                    finding(
                        'workload.size.' + name, 'policy_rejection'
                        if limit and value > limit else 'unknown', resource,
                        'Operator-measured ' + name + ': ' + str(value) +
                        '; supplied target limit: ' + str(limit) +
                        '. Measurement and target configuration are not verified.',
                        'Reduce the affected payload or tune only the corresponding supported limit within memory/concurrency constraints.',
                        'Exercise the exact upload, extracted spec, update body, parameter or metrics boundary on the candidate.'
                    ))
            missing = [name for name in SIZE_FIELDS if name not in observed]
            if missing:
                findings.append(
                    finding(
                        'workload.size', 'unknown', resource,
                        'Original boundary byte counts unavailable: ' +
                        ', '.join(missing) +
                        '. Stored JSON serialization is not the original compressed upload or update request.',
                        'Measure original upload/extracted bytes and representative update, parameter and metrics payloads; supply resource_evidence.',
                        'Test below/at/above each applicable target limit, including compressed archives and object-store reads.'
                    ))
            if shape == 'legacy_argo':
                findings.append(
                    finding(
                        'workload.cache', 'operational_impact', resource,
                        'Legacy workload may experience a cold namespace-scoped cache; inventory cannot establish hits, ownership or recomputation cost.',
                        'Budget a cold run, verify scoped cache warming and remove temporary legacy audit mode after migration.',
                        'Measure representative task duration/cost and confirm same-namespace hits plus cross-namespace misses.'
                    ))
    return _bounded_result(
        findings, summary, truncated=len(findings) > MAX_FINDINGS)


def _bounded_result(findings, summary, truncated):
    if truncated:
        findings = findings[:MAX_FINDINGS]
        findings.append(
            finding(
                'workload.coverage', 'unknown', 'installation',
                'Migration finding budget exhausted; remaining resources and controls were not assessed.',
                'Split inventory into smaller explicit scopes and rerun all partitions.',
                'Require a report without truncation for each intended scope.'))
        summary['assessment_truncated'] = True
    return findings, dict(summary)


def coverage(findings):
    controls = {}
    for item in findings:
        counts = controls.setdefault(item['rule'], Counter())
        counts[item['status']] += 1
    return [
        dict(
            control=name,
            coverage='unassessed'
            if name == 'coverage.unassessed' else 'partial',
            confidence='unknown'
            if name == 'coverage.unassessed' else 'conditional_evidence',
            counts=dict(counts)) for name, counts in sorted(controls.items())
    ]
