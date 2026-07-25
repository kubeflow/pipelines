#!/usr/bin/env python3
"""Select superseded runtime base-image cache IDs for deletion.

The caller supplies paginated output from GitHub's repository cache API. The
current cache must be visible on the trusted ref before this script emits any
IDs, preventing a delayed or failed cache save from deleting the last usable
generation.
"""

import argparse
import json
import sys
from typing import Any, Iterable


CACHE_KEY_PREFIX = 'runtime-base-images-'


def _cache_entries(pages: Any) -> Iterable[dict[str, Any]]:
    if isinstance(pages, dict):
        pages = [pages]
    if not isinstance(pages, list):
        raise ValueError('Expected a cache API response object or list of pages')

    for page in pages:
        if not isinstance(page, dict):
            raise ValueError('Each cache API page must be an object')
        entries = page.get('actions_caches')
        if not isinstance(entries, list):
            raise ValueError('Each cache API page must contain actions_caches')
        yield from entries


def cache_ids_to_delete(
    pages: Any,
    current_key: str,
    current_ref: str,
) -> list[int]:
    """Return IDs for every prefixed cache except the current trusted cache."""
    caches = list(_cache_entries(pages))
    current_cache_exists = any(
        cache.get('key') == current_key and cache.get('ref') == current_ref
        for cache in caches
    )
    if not current_cache_exists:
        raise ValueError(
            f'Current cache {current_key!r} was not found on {current_ref!r}'
        )

    cache_ids = {
        cache['id']
        for cache in caches
        if isinstance(cache.get('key'), str)
        and cache['key'].startswith(CACHE_KEY_PREFIX)
        and (cache['key'] != current_key or cache.get('ref') != current_ref)
    }
    return sorted(cache_ids)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--current-key', required=True)
    parser.add_argument('--current-ref', required=True)
    args = parser.parse_args()

    try:
        pages = json.load(sys.stdin)
        cache_ids = cache_ids_to_delete(
            pages,
            current_key=args.current_key,
            current_ref=args.current_ref,
        )
    except (KeyError, TypeError, ValueError, json.JSONDecodeError) as error:
        print(f'Cannot select caches for deletion: {error}', file=sys.stderr)
        return 1

    for cache_id in cache_ids:
        print(cache_id)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
