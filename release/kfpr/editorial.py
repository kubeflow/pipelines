"""Advisory editorial checks for release notes, without publication gates."""

import re

_EDITORIAL_HEADINGS = {
    'highlights',
    'action required',
    'compatibility',
    'known issues',  # Retain checks for existing previews.
    'known limitations',
    'bug fixes',
    'security improvements',
    'availability and resilience',
    'installation',
    'prepare',
    'roll out',
    'verify',
    'custom configurations',
}
_GUIDANCE_HEADINGS = {'breaking changes', 'deprecations'}
_FENCE = re.compile(r'^\s*(`{3,}|~{3,})(.*)$')
_HEADING = re.compile(r'^ {0,3}(#{1,6})\s+(.+?)\s*#*\s*$')
_BULLET = re.compile(r'^(\s*)[-*+]\s+(.+)')
_PLACEHOLDER = re.compile(r'\b(?:TODO|TBD|FIXME)\b')
_MIGRATION_LINK = re.compile(
    r'\[[^\]]*(?:migrat|upgrad)[^\]]*\]\([^)]+\)|https?://[^\s<>)]*(?:migrat|upgrad)[^\s<>)]*',
    re.IGNORECASE,
)


_PRODUCTS = {'backend', 'sdk', 'upgrade'}
_VERSION_HEADING = re.compile(
    r'^(?:version\s+|release\s+)?(?:sdk-)?v?\d+\.\d+\.\d+(?:[-+][\w.-]+)?(?:\s.*)?$',
    re.IGNORECASE,
)
_GENERATED_BLOCK = re.compile(
    r'<!--\s*kfpr:generated:(credits|contributors|assets):start\s*-->'
    r'.*?<!--\s*kfpr:generated:\1:end\s*-->',
    re.DOTALL | re.IGNORECASE,
)


def editorial_scaffold(product: str = 'backend') -> str:
    """Prompt product-specific decisions without asserting release facts."""
    if product not in _PRODUCTS:
        raise ValueError(
            'Choose backend, sdk, or upgrade for editorial guidance.'
        )
    common = [
        (
            'Highlights',
            'Summarize balanced user-visible benefits across the complete release range, not only recent fixes. Link supporting changes; do not turn the commit inventory into a work log.',
        ),
        (
            'Bug fixes',
            'Select consequential fixes and explain the user-visible symptom and resulting behavior. Link evidence; avoid repeating highlights.',
        ),
    ]
    if product == 'upgrade':
        sections = [
            (
                'Prepare',
                'Identify applicable backups, compatibility checks, and custom configurations. Distinguish standard installations from custom overlays, storage, roles, and clients.',
            ),
            (
                'Roll out',
                'Describe the normal deployment process and expected interruptions. Require special ordering or cutover only when the released implementation requires it.',
            ),
            (
                'Verify',
                'List representative workload and integration checks, with recovery references for failures. Do not substitute release-process checks for operator actions.',
            ),
            (
                'Compatibility',
                'Describe confirmed behavior changes and affected configurations; link detailed references instead of duplicating implementation history.',
            ),
            (
                'Custom configurations',
                'Identify additional checks for custom overlays, storage, roles, and clients without imposing them on standard installations.',
            ),
            (
                'Known limitations',
                'Explain retained limitations and actionable workarounds using verified evidence.',
            ),
        ]
    else:
        sections = common
        if product == 'backend':
            sections += [
                (
                    'Security improvements',
                    'Describe verified authorization and isolation improvements with their limits. Distinguish scoped fixes from complete security guarantees.',
                ),
                (
                    'Availability and resilience',
                    'Describe verified recovery, restart, and multi-replica improvements with their limits. Do not imply complete high availability or zero-downtime upgrades without evidence.',
                ),
            ]
        else:
            sections += [
                (
                    'Installation',
                    'Retain generated installation commands for the target packaging layout. Explain confirmed interpreter, dependency, backend, and compiled-component compatibility.',
                ),
            ]
        sections += [
            (
                'Action required',
                'Explain before/after behavior, who is affected, required action, and migration links. State removal versions or dates only when confirmed.',
            ),
            (
                'Compatibility',
                'Distinguish tested combinations from supported combinations. Identify unevaluated combinations without implying support; use target-release facts.',
            ),
            (
                'Known limitations',
                'List confirmed limitations and workarounds or tracking links. Remove unused sections after review; do not invent claims.',
            ),
        ]
    return ''.join(
        f'## {heading}\n\n<!-- TODO: {prompt} -->\n\n'
        for heading, prompt in sections
    )


def _outside_fences(text: str) -> tuple[list[str], set[int]]:
    """Mask code while remembering that real code can fill a section."""
    lines = []
    code_content = set()
    marker = ''
    for number, line in enumerate(text.splitlines(), 1):
        fence = _FENCE.match(line)
        if marker:
            if (
                fence
                and fence[1][0] == marker[0]
                and len(fence[1]) >= len(marker)
                and not fence[2].strip()
            ):
                marker = ''
            elif line.strip():
                code_content.add(number)
            lines.append('')
        elif fence and (fence[1][0] == '~' or '`' not in fence[2]):
            marker = fence[1]
            lines.append('')
        else:
            lines.append(line)
    return lines, code_content


def validate_notes(text: str, product: str | None = None) -> list[str]:
    """Return heuristic editorial warnings; callers decide how to display
    them."""
    if product is not None and product not in _PRODUCTS:
        raise ValueError(
            'Choose backend, sdk, or upgrade for editorial checks.'
        )
    lines, code_content = _outside_fences(text)
    warnings = [
        f'Line {number}: unfinished TODO/TBD/FIXME placeholder; resolve or remove it.'
        for number, line in enumerate(lines, 1)
        if _PLACEHOLDER.search(line)
    ]
    # Comments still trigger placeholders above but cannot satisfy prose checks.
    visible = re.sub(
        r'<!--.*?(?:-->|$)',
        lambda match: re.sub(r'[^\n]', ' ', match[0]),
        '\n'.join(lines),
        flags=re.DOTALL,
    ).split('\n')
    headings = []
    seen = {}
    category = None
    pending = None

    def finish_entry():
        if pending is not None and not pending['guidance']:
            warnings.append(
                f'Line {pending["line"]}: {pending["category"]} entry may need migration guidance; '
                'add an indented explanation of before/after behavior and required action, or a migration link.'
            )

    for number, line in enumerate(visible, 1):
        heading = _HEADING.match(line)
        if heading:
            finish_entry()
            pending = None
            level, title = len(heading[1]), heading[2].strip().casefold()
            headings.append((number, level, title))
            if title in _GUIDANCE_HEADINGS:
                category = (level, title)
            elif category and level <= category[0]:
                category = None
            continue
        bullet = _BULLET.match(line)
        if bullet:
            entry = re.sub(r'\s+', ' ', bullet[2]).strip().casefold()
            if entry in seen:
                warnings.append(
                    f'Line {number}: duplicate bullet entry (also on line {seen[entry]}).'
                )
            else:
                seen[entry] = number
            if pending and len(bullet[1]) > pending['indent']:
                pending['guidance'] = True
                continue
            finish_entry()
            pending = None
            if category:
                pending = {
                    'line': number,
                    'indent': len(bullet[1]),
                    'category': category[1],
                    'guidance': bool(_MIGRATION_LINK.search(line)),
                }
        elif pending and line.strip():
            indent = len(line) - len(line.lstrip())
            if indent > pending['indent'] or _MIGRATION_LINK.search(line):
                pending['guidance'] = True
    finish_entry()
    for index, (number, level, title) in enumerate(headings):
        if title not in _EDITORIAL_HEADINGS:
            continue
        end = next(
            (
                other_number
                for other_number, other_level, _ in headings[index + 1 :]
                if other_level <= level
            ),
            len(visible) + 1,
        )
        if not any(
            (
                visible[position - 1].strip()
                and not _HEADING.match(visible[position - 1])
            )
            or position in code_content
            for position in range(number + 1, end)
        ):
            warnings.append(
                f'Line {number}: editorial section "{title}" is empty; complete it or remove the heading.'
            )
    version_headings = [
        (number, title)
        for number, _, title in headings
        if _VERSION_HEADING.match(title)
    ]
    if len(version_headings) > 1:
        warnings.append(
            'Multiple version-history headings may include prior-release content; '
            'keep this release focused on its selected range.'
        )
    prose = _GENERATED_BLOCK.sub('', '\n'.join(lines))
    prose = re.sub(r'<!--.*?(?:-->|$)', '', prose, flags=re.DOTALL)
    words = len(re.findall(r"\b[\w]+(?:['’-][\w]+)*\b", prose))
    limit = 1800 if product == 'upgrade' else 1200
    if words > limit:
        warnings.append(
            f'Editorial prose contains {words} words (guideline: {limit}); '
            'summarize key changes and link detailed references.'
        )
    if product in {'backend', 'sdk'}:
        titles = {title for _, _, title in headings}
        for section, aliases in (
            ('Bug fixes', {'bug fixes'}),
            ('Contributors', {'contributors', 'new contributors', 'credits'}),
            ('Assets', {'assets', 'release assets'}),
        ):
            if not titles & aliases:
                warnings.append(
                    f'Missing {section} section; review applicable content and '
                    'preserve generated credits/assets without inventing entries.'
                )
    return warnings
