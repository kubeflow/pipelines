"""Match public repository advisories to a release and its user-facing notes."""

import json
import re
from dataclasses import dataclass

from markdown_it import MarkdownIt

from .github_api import REPO
from .packages import LEGACY_PACKAGES
from .runner import CommandRunner

_SDK_DISTRIBUTIONS = {name for _, name, _ in LEGACY_PACKAGES}
_ADVISORY_BASE = f'https://github.com/{REPO}/security/advisories/'
_GHSA = r'GHSA-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}'
# Match fixed release identifiers, not upper bounds, release candidates or
# later versions that simply inherit the patch.
_FIXED_RELEASE = re.compile(r'(?:>=|==|=)?\s*v?(\d+\.\d+\.\d+)\s*')
_BARE_LINK = re.compile(
    rf'(?:^|[\s(<]){re.escape(_ADVISORY_BASE)}({_GHSA})/?'
    r'(?=[)>,;\s]|\.(?:\s|$)|$)'
)


@dataclass(frozen=True)
class ReleaseAdvisory:
    ghsa_id: str
    summary: str
    products: frozenset[str]

    @property
    def url(self) -> str:
        return _ADVISORY_BASE + self.ghsa_id


def _patches_version(patched_versions: str | None, version: str) -> bool:
    for entry in re.split(r'[,;\n]|\|\|', patched_versions or ''):
        match = _FIXED_RELEASE.fullmatch(entry.strip())
        if match and match[1] == version:
            return True
    return False


def fetch_release_advisories(
    runner: CommandRunner, version: str
) -> list[ReleaseAdvisory]:
    """Read all published pages and match affected packages individually."""
    raw = runner.capture(
        [
            'gh',
            'api',
            '--paginate',
            '--slurp',
            f'repos/{REPO}/security-advisories?state=published&per_page=100',
        ]
    )
    pages = json.loads(raw)
    if not isinstance(pages, list) or not pages:
        raise ValueError('Missing published security advisory response pages.')
    advisories = {}
    for page in pages:
        if not isinstance(page, list):
            raise ValueError(
                'Invalid published security advisory response page.'
            )
        for row in page:
            if (
                row['state'] != 'published'
                or not row['published_at']
                or row.get('withdrawn_at')
            ):
                continue
            products = set()
            for affected in row['vulnerabilities']:
                if not _patches_version(affected['patched_versions'], version):
                    continue
                package = affected['package']
                name = re.sub(r'[-_.]+', '-', (package['name'] or '').lower())
                sdk = (
                    package['ecosystem'] == 'pip' and name in _SDK_DISTRIBUTIONS
                )
                products.add('sdk' if sdk else 'backend')
            if products:
                ghsa_id = row['ghsa_id']
                if not re.fullmatch(_GHSA, ghsa_id):
                    raise ValueError('Invalid published security advisory ID.')
                advisories[ghsa_id] = ReleaseAdvisory(
                    ghsa_id, row['summary'], frozenset(products)
                )
    return [advisories[key] for key in sorted(advisories)]


def _linked_advisories(text: str) -> set[str]:
    """Check rendered link destinations and GitHub's bare URLs in ordinary prose."""
    ids = set()
    for block in MarkdownIt('commonmark').parse(text):
        if block.type != 'inline':
            continue
        in_link = False
        for token in block.children or []:
            if token.type == 'link_open':
                in_link = True
                href = str(token.attrGet('href')).split('#', 1)[0].rstrip('/')
                if href.startswith(_ADVISORY_BASE):
                    ids.add(href.removeprefix(_ADVISORY_BASE))
            elif token.type == 'link_close':
                in_link = False
            elif token.type == 'text' and not in_link:
                ids.update(_BARE_LINK.findall(token.content))
    return ids


def validate_advisory_coverage(
    advisories: list[ReleaseAdvisory], notes: dict[str, str]
) -> None:
    """Require links on the release-wide backend and applicable SDK pages."""
    missing = []
    for product, text in notes.items():
        linked = _linked_advisories(text)
        for advisory in advisories:
            if product == 'backend' or product in advisory.products:
                if advisory.ghsa_id not in linked:
                    missing.append(f'{product}: {advisory.url}')
    if missing:
        raise ValueError(
            'Missing published advisory links:\n' + '\n'.join(missing)
        )
