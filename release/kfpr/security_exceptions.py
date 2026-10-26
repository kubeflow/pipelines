"""Load explicit, expiring exceptions without broadening scanner coverage."""

import hashlib
import json
import re
from datetime import date, datetime, timezone
from pathlib import Path

_FIELDS = {
    'scope',
    'artifact',
    'vulnerability',
    'package',
    'installed_version',
    'justification',
    'expires',
}
_IDENTITY = (
    'scope',
    'artifact',
    'package',
    'vulnerability',
    'installed_version',
)


def validate_exceptions(document, *, now: datetime | None = None) -> dict:
    """Validate exact identities and expiration against the current UTC day."""
    if (
        not isinstance(document, dict)
        or set(document) != {'schema', 'exceptions'}
        or type(document['schema']) is not int
        or document['schema'] != 1
        or not isinstance(document['exceptions'], list)
    ):
        raise ValueError(
            'Invalid security exception document; expected schema 1 and exceptions list'
        )
    today = (now or datetime.now(timezone.utc)).astimezone(timezone.utc).date()
    seen = set()
    validated = []
    for entry in document['exceptions']:
        if not isinstance(entry, dict) or set(entry) != _FIELDS:
            raise ValueError('Invalid security exception fields')
        if any(
            not isinstance(value, str) or not value.strip()
            for value in entry.values()
        ):
            raise ValueError(
                'Security exception fields must be nonempty strings'
            )
        if entry['scope'] not in ('source', 'python', 'image'):
            raise ValueError('Invalid security exception scope')
        if any(
            any(character in entry[key] for character in '*?[]')
            for key in _IDENTITY
        ):
            raise ValueError(
                'Security exceptions must use exact identities, not wildcards'
            )
        if entry['scope'] == 'image' and not re.fullmatch(
            r'sha256:[0-9a-f]{64}', entry['artifact']
        ):
            raise ValueError(
                'Image exception artifact must be an immutable image ID'
            )
        if entry['scope'] == 'python' and not re.fullmatch(
            r'[0-9]+\.[0-9]+', entry['artifact']
        ):
            raise ValueError(
                'Python exception artifact must be a Python major.minor version'
            )
        if not re.fullmatch(r'\d{4}-\d{2}-\d{2}', entry['expires']):
            raise ValueError('Security exception expiry must be YYYY-MM-DD')
        try:
            expires = date.fromisoformat(entry['expires'])
        except ValueError as error:
            raise ValueError(
                'Invalid security exception expiry date'
            ) from error
        if expires < today:
            raise ValueError(
                f'Security exception expired: {entry["vulnerability"]} ({entry["expires"]})'
            )
        identity = tuple(entry[key] for key in _IDENTITY)
        if identity in seen:
            raise ValueError('Duplicate security exception identity')
        seen.add(identity)
        validated.append(dict(entry))
    return {'schema': 1, 'exceptions': validated}


def _unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f'Duplicate security exception JSON key: {key}')
        result[key] = value
    return result


def load_exceptions(path: Path, *, now: datetime | None = None) -> dict:
    """Read and validate a security exception file, including current
    expiry."""
    return validate_exceptions(
        json.loads(Path(path).read_text(), object_pairs_hook=_unique_object),
        now=now,
    )


def exception_digest(document: dict) -> str:
    """Hash the complete document independently of JSON whitespace/key
    order."""
    canonical = json.dumps(
        document, sort_keys=True, separators=(',', ':'), ensure_ascii=False
    )
    return hashlib.sha256(canonical.encode('utf-8')).hexdigest()
