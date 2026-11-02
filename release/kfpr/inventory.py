"""Exact-range evidence and reviewed facts for release editorial work.

This module does not infer compatibility or turn commit mentions into
features. Editors select benefits from the complete inventory before
rendering documents.
"""

import html
import json
import re
from pathlib import Path
from urllib.parse import quote

from .editorial import editorial_scaffold
from .ranges import ReleaseRange, _git, read_commits

REPOSITORY = 'https://github.com/kubeflow/pipelines'
PRODUCTS = ('backend', 'sdk', 'upgrade')
SECTIONS = {
    'highlights': 'Highlights',
    'bug_fixes': 'Bug fixes',
    'security': 'Security improvements',
    'resilience': 'Availability and resilience',
    'compatibility': 'Compatibility',
    'limitations': 'Known limitations',
    'prepare': 'Prepare',
    'rollout': 'Roll out',
    'verify': 'Verify',
    'custom': 'Custom configurations',
    'action': 'Action required',
}
COVERAGE = (
    'features',
    'bug_fixes',
    'security',
    'resilience',
    'compatibility',
    'operator_actions',
)
_NESTED_SUBJECT = re.compile(
    r'^(?:\* |- )?((?:feat|fix|perf|security|revert|docs|chore|test|build|ci)'
    r'(?:\([^\n)]*\))?!?: .+)$',
    re.IGNORECASE,
)


def nested_subjects(message: str) -> list[dict]:
    """Expose squash/backport subjects with body line numbers, excluding
    fences."""
    result = []
    marker = ''
    for number, line in enumerate(message.splitlines()[1:], 2):
        fence = re.match(r'^ {0,3}(`{3,}|~{3,})(.*)$', line)
        if marker:
            if (
                fence
                and fence[1][0] == marker[0]
                and len(fence[1]) >= len(marker)
                and not fence[2].strip()
            ):
                marker = ''
            continue
        if fence:
            marker = fence[1]
            continue
        match = _NESTED_SUBJECT.match(line)
        if match:
            result.append({'line': number, 'subject': match[1]})
    return result


def build_inventory(root: Path, selected: ReleaseRange, version: str) -> dict:
    """Inventory every selected commit, including nonconventional changes."""
    commits = read_commits(root, selected)
    authors = _git(
        root,
        'log',
        '-z',
        '--format=%H%x00%an%x00%(trailers:key=Co-authored-by,valueonly,separator=%x1f)',
        f'{selected.base_sha}..{selected.target_sha}',
        '--',
    ).split('\0')
    by_sha = {
        authors[i]: (authors[i + 1], authors[i + 2])
        for i in range(0, len(authors) - 1, 3)
    }
    return {
        'schema_version': 1,
        'version': version,
        'range': selected.as_dict(),
        'facts': [],
        'coverage': {category: '' for category in COVERAGE},
        'changes': [
            {
                'sha': commit.sha,
                'author': by_sha[commit.sha][0],
                'coauthors': sorted(
                    set(
                        match[1].strip()
                        for value in by_sha[commit.sha][1].split('\x1f')
                        if (match := re.fullmatch(r'(.+?)\s*<[^>]+>\s*', value))
                    )
                ),
                'subject': commit.message.splitlines()[0],
                'message': commit.message,
                'nested_subjects': nested_subjects(commit.message),
                'decision': 'pending',
                'reason': '',
                'notes': [],
            }
            for commit in commits
        ],
    }


def validate_inventory(inventory: dict) -> None:
    """Reject malformed review inputs instead of silently dropping coverage."""
    if not isinstance(inventory, dict) or inventory.get('schema_version') != 1:
        raise ValueError(
            'Unsupported inventory schema; generate an inventory with prepare-editorial.'
        )
    if not isinstance(inventory.get('version'), str) or not re.fullmatch(
        r'\d+\.\d+\.\d+(?:[-.][A-Za-z0-9.-]+)?', inventory['version']
    ):
        raise ValueError(
            'Inventory version must be a release version such as 2.18.0.'
        )
    selected = inventory.get('range', {})
    if (
        not isinstance(selected, dict)
        or set(selected) != {'base_ref', 'target_ref', 'base_sha', 'target_sha'}
        or any(
            not isinstance(selected.get(key), str) or not selected[key]
            for key in ('base_ref', 'target_ref', 'base_sha', 'target_sha')
        )
    ):
        raise ValueError('Inventory needs its original resolved release range.')
    if any(
        not re.fullmatch(r'[0-9a-f]{40}|[0-9a-f]{64}', selected[key])
        for key in ('base_sha', 'target_sha')
    ):
        raise ValueError('Inventory range must contain full commit SHAs.')
    changes = inventory.get('changes')
    if not isinstance(changes, list):
        raise ValueError('Inventory changes must be a list.')
    seen = set()
    for change in changes:
        if (
            not isinstance(change, dict)
            or not isinstance(change.get('sha'), str)
            or not re.fullmatch(r'[0-9a-f]{40}|[0-9a-f]{64}', change['sha'])
            or change['sha'] in seen
        ):
            raise ValueError(
                'Inventory changes require unique full commit SHAs.'
            )
        seen.add(change['sha'])
        if any(
            not isinstance(change.get(key), str)
            for key in ('subject', 'message', 'author', 'reason')
        ):
            raise ValueError(
                'Inventory changes need their evidence and review reason.'
            )
        if change.get('decision') not in ('pending', 'include', 'omit'):
            raise ValueError(
                'Choose pending, include, or omit for each change.'
            )
        if change['decision'] == 'omit' and not change['reason'].strip():
            raise ValueError(
                'Explain why an omitted change needs no release coverage.'
            )
        _validate_entries(change.get('notes'))
        if change['decision'] == 'include' and not change['notes']:
            raise ValueError(
                'Included changes require at least one reviewed note.'
            )
    _validate_entries(inventory.get('facts'))
    for fact in inventory['facts']:
        if not isinstance(fact.get('commit'), str) or fact[
            'commit'
        ] not in seen | {
            selected['base_sha'],
            selected['target_sha'],
        }:
            raise ValueError(
                'Shared facts need a commit from this inventory or its release boundaries.'
            )
    coverage = inventory.get('coverage')
    if not isinstance(coverage, dict) or any(
        not isinstance(coverage.get(key), str) for key in COVERAGE
    ):
        raise ValueError(
            'Inventory needs the coverage checklist; record covered or intentionally omitted categories.'
        )


def _validate_entries(entries: object) -> None:
    if not isinstance(entries, list):
        raise ValueError('Notes and shared facts must be lists.')
    for entry in entries:
        if (
            not isinstance(entry, dict)
            or not isinstance(entry.get('section'), str)
            or entry['section'] not in SECTIONS
            or not isinstance(entry.get('text'), str)
            or not entry['text'].strip()
        ):
            raise ValueError(
                'Each note needs a known section and nonempty text.'
            )
        products = entry.get('products')
        if (
            not isinstance(products, list)
            or not products
            or any(product not in PRODUCTS for product in products)
        ):
            raise ValueError(
                'Each note needs backend, sdk, or upgrade products.'
            )
        if re.search(r'(?m)^#{1,6} |<!-- kfpr:generated:', entry['text']):
            raise ValueError(
                'Write note prose without section headings or generated-block markers.'
            )


def verify_evidence(root: Path, inventory: dict) -> None:
    """Prevent stale/mutated inventories and master-only additions from
    rendering."""
    validate_inventory(inventory)
    selected = ReleaseRange(**inventory['range'])
    expected = build_inventory(root, selected, inventory['version'])
    evidence = (
        'sha',
        'author',
        'coauthors',
        'subject',
        'message',
        'nested_subjects',
    )
    if [
        {key: change.get(key) for key in evidence}
        for change in inventory['changes']
    ] != [
        {key: change[key] for key in evidence} for change in expected['changes']
    ]:
        raise ValueError(
            'Inventory evidence differs from the release range. Generate a fresh inventory and reapply review decisions.'
        )


def review_warnings(inventory: dict) -> list[str]:
    pending = sum(
        change['decision'] == 'pending' for change in inventory['changes']
    )
    warnings = (
        [f'{pending} commits still need a coverage decision.']
        if pending
        else []
    )
    warnings += [
        f'Review {key} coverage.'
        for key, value in inventory['coverage'].items()
        if not value.strip()
    ]
    if not inventory['facts']:
        warnings.append(
            'Record verified compatibility and limitations in shared facts before publication.'
        )
    return warnings


def release_scaffolding(inventory: dict, product: str) -> str:
    """Generate credits and source assets from evidence, never guess GitHub
    handles."""
    tag = ('sdk-' if product == 'sdk' else '') + inventory['version']
    authors = {}
    for change in inventory['changes']:
        authors.setdefault(change['author'], change['sha'])
        for name in change.get('coauthors', []):
            authors.setdefault(name, change['sha'])
    credits = ', '.join(
        f'[{html.escape(name).replace("[", "&#91;").replace("]", "&#93;")}]({REPOSITORY}/commit/{sha})'
        for name, sha in sorted(
            authors.items(), key=lambda item: item[0].casefold()
        )
    )
    if not credits:
        credits = 'No commits in the selected release range.'
    encoded = quote(tag, safe='')
    return (
        '<!-- kfpr:generated:credits:start -->\n## Contributors\n\n'
        '<details>\n<summary>Release contributors</summary>\n\n'
        + credits
        + '\n\n</details>\n<!-- kfpr:generated:credits:end -->\n\n'
        '<!-- kfpr:generated:assets:start -->\n## Assets\n\n'
        f'Source archives (available after tag publication): [ZIP]({REPOSITORY}/archive/refs/tags/{encoded}.zip) · '
        f'[tar.gz]({REPOSITORY}/archive/refs/tags/{encoded}.tar.gz).\n\n'
        f'See the [release assets]({REPOSITORY}/releases/tag/{encoded}) for published downloads.\n'
        '<!-- kfpr:generated:assets:end -->\n'
    )


def render_document(inventory: dict, product: str) -> str:
    """Render only reviewed prose, with one shared source for cross-product
    facts."""
    validate_inventory(inventory)
    if product not in PRODUCTS:
        raise ValueError('Choose backend, sdk, or upgrade.')
    entries = list(inventory['facts'])
    for change in inventory['changes']:
        if change['decision'] == 'include':
            for note in change['notes']:
                entries.append(
                    note
                    | {
                        'text': note['text']
                        + f' [commit]({REPOSITORY}/commit/{change["sha"]})'
                    }
                )
    sections = {}
    for entry in entries:
        if product in entry['products']:
            sections.setdefault(entry['section'], []).append(entry['text'])
    # The scaffold remains as a coverage checklist until editors complete it.
    text = editorial_scaffold(product)
    for key, title in SECTIONS.items():
        if key not in sections:
            continue
        body = '\n'.join('- ' + item for item in sections[key]) + '\n\n'
        pattern = rf'(?ms)^## {re.escape(title)}\n.*?(?=^## |\Z)'
        replacement = f'## {title}\n\n{body}'
        if re.search(pattern, text):
            text = re.sub(pattern, lambda _: replacement, text)
        else:
            text += replacement
    if product != 'upgrade':
        selected = ReleaseRange(**inventory['range'])
        text += f'**Full changelog:** {selected.compare_url}\n\n'
        text += release_scaffolding(inventory, product)
    return text


def write_bundle(root: Path, inventory: dict, output: Path) -> None:
    """Write a new review bundle without overwriting a previous editorial
    pass."""
    verify_evidence(root, inventory)
    documents = {
        product: render_document(inventory, product) for product in PRODUCTS
    }
    output.mkdir(parents=True, exist_ok=False)
    (output / 'inventory.json').write_text(
        json.dumps(inventory, indent=2) + '\n', encoding='utf-8'
    )
    for product, text in documents.items():
        (output / f'{product}.md').write_text(text, encoding='utf-8')
