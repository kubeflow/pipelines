"""Render the backend and SDK release pages and check their shared coverage."""

from pathlib import Path

from rich.console import Console
from rich.markdown import Markdown
from rich.table import Table

from .advisory_coverage import (
    fetch_release_advisories,
    validate_advisory_coverage,
)
from .context import ReleaseContext
from .editorial import publication_blockers
from .github_api import REPO

_PRODUCT_NAMES = {'backend': 'Backend', 'sdk': 'SDK'}


def _render_pages(sources: dict[str, str], notes: dict[str, str]) -> None:
    console = Console()
    for product, source in sources.items():
        print(f'{_PRODUCT_NAMES[product]} release notes: {source}')
    if len(notes) > 1 and console.width >= 120:
        pages = Table(expand=True, show_lines=False)
        for product in notes:
            pages.add_column(_PRODUCT_NAMES[product], ratio=1)
        pages.add_row(*(Markdown(body) for body in notes.values()))
        console.print(pages)
    else:
        for product, body in notes.items():
            console.rule(f'{_PRODUCT_NAMES[product]} release notes')
            console.print(Markdown(body))


def show_review(
    context: ReleaseContext,
    paths: dict[str, Path | None],
    *,
    display: bool = True,
) -> list[str]:
    """Read the current pages and public advisories on every review.

    Completed sibling releases are read from GitHub, so a checkpoint resumed
    after public disclosure also checks what users can actually read.
    """
    notes: dict[str, str] = {}
    sources: dict[str, str] = {}
    issues: list[str] = []
    if not paths:
        return ['Select at least one release product to review.']
    for product, path in paths.items():
        if path is None:
            tag = (
                context.metadata.sdk_tag
                if product == 'sdk'
                else context.metadata.tag
            )
            sources[product] = f'https://github.com/{REPO}/releases/tag/{tag}'
            notes[product] = context.runner.capture(
                [
                    'gh',
                    'release',
                    'view',
                    tag,
                    '--repo',
                    REPO,
                    '--json',
                    'body,isDraft',
                    '--jq',
                    'if .isDraft then error("Release is still a draft") else .body end',
                ],
                cwd=context.root,
            )
        else:
            sources[product] = str(path)
            notes[product] = path.read_text(encoding='utf-8')
        issues.extend(
            f'{product}: {problem}'
            for problem in publication_blockers(notes[product])
        )

    if display:
        _render_pages(sources, notes)

    advisories = fetch_release_advisories(context.runner, context.metadata.tag)
    selected = [
        advisory
        for advisory in advisories
        if 'backend' in notes or 'sdk' in advisory.products
    ]
    if display:
        backend = sum('backend' in advisory.products for advisory in selected)
        sdk = sum('sdk' in advisory.products for advisory in selected)
        noun = 'advisory' if len(selected) == 1 else 'advisories'
        print(
            f'{len(selected)} published {noun} for the selected release pages ({backend} backend, {sdk} SDK).'
        )
        for advisory in selected:
            print(f'{advisory.ghsa_id}: {advisory.summary} — {advisory.url}')
    complete_coverage = True
    for product, body in notes.items():
        try:
            validate_advisory_coverage(advisories, {product: body})
        except ValueError as error:
            issues.append(
                f'{error}\nUpdate the {product} notes at {sources[product]} and retry.'
            )
            complete_coverage = False
    if display and complete_coverage:
        print('All required public advisory links are present.')
    return issues
