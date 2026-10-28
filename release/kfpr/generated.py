"""Keep generated release credits/downloads separate from editable prose."""

import re

_NAMES = ('credits', 'assets')


def _validate_blocks(text: str) -> None:
    active = None
    seen = set()
    for name, boundary in re.findall(
        r'<!-- kfpr:generated:(credits|assets):(start|end) -->', text
    ):
        if boundary == 'start':
            if active or name in seen:
                raise ValueError(
                    'Malformed generated markers: blocks must be unique and cannot nest. Restore complete blocks and retry.'
                )
            active = name
            seen.add(name)
        elif active != name:
            raise ValueError(
                'Malformed generated markers: unmatched or crossed end marker. Restore complete blocks and retry.'
            )
        else:
            active = None
    if active:
        raise ValueError(
            'Malformed generated markers: missing end marker. Restore the complete block and retry.'
        )


def restore_generated_sections(text: str, generated: str) -> str:
    """Restore deleted/edited managed blocks, leaving editorial prose untouched.

    Legacy previews without managed blocks are unchanged. Ambiguous markers
    fail closed so malformed edits cannot cause unrelated prose to be removed.
    """
    if not isinstance(generated, str):
        raise ValueError(
            'Generated baseline must be Markdown text; restore the preview provenance before retrying.'
        )
    _validate_blocks(generated)
    _validate_blocks(text)
    for name in _NAMES:
        start = f'<!-- kfpr:generated:{name}:start -->'
        end = f'<!-- kfpr:generated:{name}:end -->'
        pattern = re.compile(
            re.escape(start) + r'.*?' + re.escape(end), re.DOTALL
        )
        expected = pattern.findall(generated)
        if not expected and start not in generated and end not in generated:
            continue
        if (
            len(expected) != 1
            or generated.count(start) != 1
            or generated.count(end) != 1
        ):
            raise ValueError(
                'Generated release section is malformed; restore the preview baseline.'
            )
        if text.count(start) == 0 and text.count(end) == 0:
            text = text.rstrip() + '\n\n' + expected[0] + '\n'
        elif (
            text.count(start) == 1
            and text.count(end) == 1
            and pattern.search(text)
        ):
            text = pattern.sub(lambda _: expected[0], text)
        else:
            raise ValueError(
                f'Malformed {name} markers; restore or remove the entire generated block and retry.'
            )
    return text
