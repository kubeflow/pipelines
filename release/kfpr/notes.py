"""Shared user-facing changelog rules for SDK and backend releases."""

import re
from collections.abc import Iterable
from dataclasses import dataclass
from typing import Literal

_REPOSITORY = 'https://github.com/kubeflow/pipelines'
_SUBJECT = re.compile(
    r'^(?P<type>[A-Za-z]+)(?:\((?P<scope>[^)]*)\))?(?P<breaking>!)?:\s*(?P<description>.+)$'
)
_BREAKING_FOOTER = re.compile(r'^BREAKING[ -]CHANGE:\s*\S', re.MULTILINE)
_DEPRECATION_FOOTER = re.compile(
    r'^(?:DEPRECATED|DEPRECATION):\s*\S', re.MULTILINE
)
_TRAILER = re.compile(
    r'^(?P<token>BREAKING CHANGE|[A-Za-z][A-Za-z0-9-]*)'
    r'(?::(?:[ \t]+|(?=\r?$))|[ \t]+#)',
    re.MULTILINE,
)
_FENCE = re.compile(r'^ {0,3}(`{3,}|~{3,})(.*)$')
_GUIDANCE_TRAILERS = {
    'BREAKING CHANGE',
    'BREAKING-CHANGE',
    'DEPRECATED',
    'DEPRECATION',
}
# Only known metadata trailers end guidance; prose labels such as Before: do not.
_METADATA_TRAILERS = {
    'signed-off-by',
    'co-authored-by',
    'reviewed-by',
    'acked-by',
    'tested-by',
    'reported-by',
    'suggested-by',
    'refs',
    'fixes',
    'closes',
    'resolves',
}
_SDK_SCOPES = {
    'sdk',
    'kfp',
    'kfp-kubernetes',
    'kubernetes',
    'kubernetes-platform',
    'kubernetes_platform',
    'kfp-server-api',
    'server-api',
}
_BACKEND_SCOPES = {'backend', 'runtime', 'frontend', 'manifests'}
_SHARED_SCOPES = {'api', 'pipeline-spec', 'kfp-pipeline-spec'}
_SCOPE_ALIASES = {
    'deployment': 'manifests',
    'cache': 'backend/cache',
    'metadata': 'backend/metadata',
    'swf': 'backend/scheduled-workflow',
    'viewer': 'frontend/viewer',
}
_GROUPS = (
    'Breaking changes',
    'Security fixes',
    'Features',
    'Performance',
    'Deprecations',
    'Bug fixes',
)


@dataclass(frozen=True)
class Commit:
    """A commit identity and its complete subject and body."""

    sha: str
    message: str


def _normalize_scope(scope: str) -> str:
    groups = []
    for group in scope.split(','):
        parts = []
        for part in group.split('/'):
            token = part.strip().lower()
            if token:
                parts.extend(_SCOPE_ALIASES.get(token, token).split('/'))
        if parts:
            groups.append('/'.join(dict.fromkeys(parts)))
    return ', '.join(groups)


def _applies_to_product(scope: str, product: str) -> bool:
    scopes = {part.strip().lower() for part in re.split(r'[/,]', scope)}
    known = scopes & (_SDK_SCOPES | _BACKEND_SCOPES | _SHARED_SCOPES)
    if product == 'all' or scopes & _SHARED_SCOPES:
        return True
    if not known:
        return product == 'backend'
    return bool(scopes & (_SDK_SCOPES if product == 'sdk' else _BACKEND_SCOPES))


def _footer_guidance(body: str) -> list[str]:
    trailers = []
    fence_marker = ''
    offset = 0
    for line in body.splitlines(keepends=True):
        fence = _FENCE.match(line.rstrip('\r\n'))
        if fence_marker:
            if (
                fence
                and fence[1][0] == fence_marker[0]
                and len(fence[1]) >= len(fence_marker)
                and not fence[2].strip()
            ):
                fence_marker = ''
        elif fence and (fence[1][0] == '~' or '`' not in fence[2]):
            fence_marker = fence[1]
        else:
            trailer = _TRAILER.match(body, offset)
            if trailer and (
                trailer.group('token') in _GUIDANCE_TRAILERS
                or trailer.group('token').lower() in _METADATA_TRAILERS
            ):
                trailers.append(trailer)
        offset += len(line)
    guidance = []
    for index, trailer in enumerate(trailers):
        if trailer.group('token') not in _GUIDANCE_TRAILERS:
            continue
        end = (
            trailers[index + 1].start()
            if index + 1 < len(trailers)
            else len(body)
        )
        detail = body[trailer.end() : end].strip()
        if detail:
            guidance.append(detail)
    return guidance


def render_notes(
    commits: Iterable[Commit],
    product: Literal['all', 'sdk', 'backend'] = 'all',
) -> str:
    """Render eligible changes in stable groups, omitting empty sections."""
    if product not in {'all', 'sdk', 'backend'}:
        raise ValueError(
            'Unknown release product; choose all, sdk, or backend.'
        )
    groups: dict[str, list[str]] = {title: [] for title in _GROUPS}
    for commit in commits:
        subject, _, body = commit.message.partition('\n')
        match = _SUBJECT.match(subject)
        if not match:
            continue
        scope = _normalize_scope(match.group('scope') or '')
        if not _applies_to_product(scope, product):
            continue
        commit_type = match.group('type').lower()
        scopes = {part.strip().lower() for part in re.split(r'[/,]', scope)}
        if (
            match.group('breaking')
            or _BREAKING_FOOTER.search(body)
            or commit_type in {'break', 'breaking'}
        ):
            group = 'Breaking changes'
        elif commit_type == 'security' or scopes & {'security', 'cve'}:
            group = 'Security fixes'
        elif (
            commit_type in {'deprecate', 'deprecated', 'deprecation'}
            or scopes & {'deprecate', 'deprecated', 'deprecation'}
            or _DEPRECATION_FOOTER.search(body)
        ):
            group = 'Deprecations'
        else:
            group = {
                'feat': 'Features',
                'perf': 'Performance',
                'fix': 'Bug fixes',
            }.get(commit_type)
        if group is None:
            continue
        description = match.group('description')
        description, pr_count = re.subn(
            r'\(#(\d+)\)$',
            lambda pr: f'([#{pr[1]}]({_REPOSITORY}/pull/{pr[1]}))',
            description,
        )
        if not pr_count:
            description += (
                f' ([{commit.sha[:7]}]({_REPOSITORY}/commit/{commit.sha}))'
            )
        prefix = f'**{scope}**: ' if scope else ''
        entry = f'* {prefix}{description}'
        for detail in _footer_guidance(body):
            entry += '\n\n' + '\n'.join(
                '  ' + line if line else '' for line in detail.splitlines()
            )
        groups[group].append(entry)
    return '\n\n'.join(
        f'## {title}\n\n' + '\n'.join(entries)
        for title, entries in groups.items()
        if entries
    )
