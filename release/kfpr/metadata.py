"""Release version metadata."""

import re
from dataclasses import dataclass
from typing import Optional

VERSION_RE = r'^(\d+)\.(\d+)\.(\d+)$'


@dataclass(frozen=True)
class ReleaseMetadata:
    """Immutable release metadata derived from version and release type."""

    release_type: str
    version: str
    major: int
    minor: int
    patch: int
    tag: str
    sdk_tag: str
    release_branch: str
    patch_branch: Optional[str]

    @classmethod
    def from_version(cls, release_type: str, version: str) -> 'ReleaseMetadata':
        """Derive release metadata from release type and version.

        Args:
          release_type: One of 'major', 'minor', or 'patch'.
          version: Semantic version string in MAJOR.MINOR.PATCH format.

        Returns:
          ReleaseMetadata instance with all derived fields.

        Raises:
          ValueError: If release_type is invalid, version format is invalid,
            or version constraints are violated (major must be X.0.0, minor must
            be X.Y.0).
        """
        if release_type not in ('major', 'minor', 'patch'):
            raise ValueError(
                f'release_type must be major, minor, or patch; got {release_type}'
            )

        match = re.match(VERSION_RE, version)
        if not match:
            raise ValueError(
                f'version must be in MAJOR.MINOR.PATCH format; got {version}'
            )

        major, minor, patch = (
            int(match.group(1)),
            int(match.group(2)),
            int(match.group(3)),
        )

        if release_type == 'major' and (minor != 0 or patch != 0):
            raise ValueError(
                f'major release version must be X.0.0; got {version}'
            )

        if release_type == 'minor' and patch != 0:
            raise ValueError(
                f'minor release version must be X.Y.0; got {version}'
            )

        tag = version
        sdk_tag = f'sdk-{version}'

        if release_type == 'major':
            release_branch = f'release-{major}.0'
            patch_branch = None
        elif release_type == 'minor':
            release_branch = f'release-{major}.{minor}'
            patch_branch = None
        else:  # patch
            release_branch = f'release-{major}.{minor}'
            patch_branch = f'release-{major}.{minor}.{patch}'

        return cls(
            release_type=release_type,
            version=version,
            major=major,
            minor=minor,
            patch=patch,
            tag=tag,
            sdk_tag=sdk_tag,
            release_branch=release_branch,
            patch_branch=patch_branch,
        )
