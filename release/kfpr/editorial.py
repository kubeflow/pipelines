"""Advisory editorial checks for release notes, without publication gates."""

import re

_EDITORIAL_HEADINGS = {
    'highlights',
    'action required',
    'compatibility',
    'known issues',
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


def editorial_scaffold() -> str:
    """Provide prompts for release editors without asserting release facts."""
    return """## Highlights

<!-- TODO: Summarize the most important user-visible changes and who benefits. -->

## Action required

<!-- TODO: For breaking or deprecated behavior, explain before/after behavior, the action users must take, and link migration steps. State removal versions or dates only when confirmed. -->

## Compatibility

<!-- TODO: State confirmed compatible versions and distinguish tested combinations from supported combinations. Identify unevaluated combinations without implying support. Describe any required upgrades. -->

## Known issues

<!-- TODO: List confirmed limitations, affected versions, and workarounds or tracking links. Remove unused sections after review; do not invent claims. -->

"""


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


def validate_notes(text: str) -> list[str]:
    """Return heuristic editorial warnings; callers decide how to display
    them."""
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
    return warnings
