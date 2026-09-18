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
"""Evaluate a named core serviceaccounts/use request against target RBAC.

This models RBAC grants only, not other authorizers, account existence,
admission, identity discovery, or whether the supplied objects will be
installed. A denial requires the caller to assert a complete RBAC-only
snapshot and complete identity (including groups). Missing or malformed
relevant evidence remains unknown.
"""

_RBAC_GROUP = 'rbac.authorization.k8s.io'


def _strings(value):
    return isinstance(value, list) and all(isinstance(v, str) for v in value)


def _subject_match(subject, user, groups, binding_namespace):
    if not isinstance(subject, dict):
        return None
    kind, name = subject.get('kind'), subject.get('name')
    if not isinstance(name, str) or not name:
        return None
    if kind in ('User', 'Group'):
        if subject.get('apiGroup') != _RBAC_GROUP:
            return None
        return name == user if kind == 'User' else name in groups
    if kind == 'ServiceAccount':
        if subject.get('apiGroup', '') != '':
            return None
        subject_namespace = subject.get('namespace', '')
        if not isinstance(subject_namespace, str):
            return None
        namespace = subject_namespace or binding_namespace
        if not isinstance(namespace, str) or not namespace:
            return None
        return user == 'system:serviceaccount:' + namespace + ':' + name
    return None


def _rule_match(rule, account):
    if not isinstance(rule, dict):
        return None
    for field in ('verbs', 'apiGroups', 'resources', 'resourceNames',
                  'nonResourceURLs'):
        if field in rule and not _strings(rule[field]):
            return None
    if not rule.get('verbs'):
        return None
    if rule.get('nonResourceURLs'):
        return False if not rule.get('resources') else None
    if not rule.get('apiGroups') or not rule.get('resources'):
        return None
    return (any(v in rule['verbs'] for v in ('use', '*')) and
            any(g in rule['apiGroups'] for g in ('', '*')) and
            any(r in rule['resources'] for r in ('serviceaccounts', '*')) and
            (not rule.get('resourceNames') or account in rule['resourceNames']))


def evaluate_use(items, user, groups, namespace, account, complete=False):
    """Return allowed, denied, or unknown for the supplied target RBAC
    evidence.

    `groups` must contain the actual caller's complete group membership; no groups
    or identities are inferred. `complete=True` asserts all applicable RBAC is
    included and RBAC is the only authorizer. A partial snapshot can prove a
    grant, but cannot prove a denial. Wildcards in resourceNames are literal.
    """
    if (not isinstance(items, list) or not _strings(groups) or not all(
            isinstance(v, str) and v for v in (user, namespace, account)) or
            not isinstance(complete, bool)):
        return 'unknown'
    roles = {}
    for item in items:
        if not isinstance(item, dict):
            continue
        kind = item.get('kind')
        metadata = item.get('metadata')
        if kind not in ('Role',
                        'ClusterRole') or not isinstance(metadata, dict):
            continue
        name = metadata.get('name')
        scope = metadata.get('namespace') if kind == 'Role' else ''
        if isinstance(name, str) and isinstance(scope, str):
            roles.setdefault((kind, scope, name), []).append(item)

    uncertain = False
    for item in items:
        if not isinstance(item, dict):
            uncertain = True
            continue
        kind = item.get('kind')
        if kind not in ('RoleBinding', 'ClusterRoleBinding'):
            continue
        metadata = item.get('metadata')
        scope = ''
        if kind == 'RoleBinding':
            if not isinstance(metadata, dict) or not isinstance(
                    metadata.get('namespace'),
                    str) or not metadata['namespace']:
                uncertain = True
                continue
            scope = metadata['namespace']
            if scope != namespace:
                continue
        subjects = item.get('subjects', [])
        if not isinstance(subjects, list):
            uncertain = True
            continue
        matches = [_subject_match(s, user, groups, scope) for s in subjects]
        if True not in matches:
            uncertain = uncertain or None in matches
            continue
        ref = item.get('roleRef')
        if (not isinstance(ref, dict) or ref.get('apiGroup') != _RBAC_GROUP or
                ref.get('kind') not in ('Role', 'ClusterRole') or
                not isinstance(ref.get('name'), str) or not ref['name'] or
            (kind == 'ClusterRoleBinding' and ref['kind'] != 'ClusterRole')):
            uncertain = True
            continue
        role_scope = scope if ref['kind'] == 'Role' else ''
        candidates = roles.get((ref['kind'], role_scope, ref['name']), [])
        if len(candidates) != 1:
            uncertain = True
            continue
        role = candidates[0]
        rules = role.get('rules', [])
        if not isinstance(rules, list):
            uncertain = True
            continue
        if role.get('aggregationRule') is not None:
            # Rendered or retained aggregate rules may be replaced by selectors.
            uncertain = True
            continue
        for rule in rules:
            match = _rule_match(rule, account)
            if match is True:
                return 'allowed'
            if match is None:
                uncertain = True
    return 'denied' if complete and not uncertain else 'unknown'
