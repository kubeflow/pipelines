"""Tests for the embedded-archive extraction helper."""

import contextlib
import io
import os
import tarfile
import tempfile
import unittest
import warnings

from kfp.dsl.templates import safe_extract

# Resolved here rather than inside a class body, where the leading double
# underscore would trigger private name mangling.
_safe_extract = getattr(safe_extract, '__kfp_safe_extract')

# The filtered path raises a tarfile.FilterError subclass; the compatibility
# path raises RuntimeError. Both are refusals.
_REFUSALS = (RuntimeError, tarfile.TarError)


@contextlib.contextmanager
def _without_extraction_filters():
    """Simulate a runtime predating the PEP 706 extraction filters."""
    missing = object()
    saved = getattr(tarfile, 'data_filter', missing)
    if saved is not missing:
        del tarfile.data_filter
    try:
        with warnings.catch_warnings():
            # The unfiltered extractall the compatibility path falls through to
            # is deprecated on the interpreters this context manager pretends
            # not to be running on.
            warnings.simplefilter('ignore', DeprecationWarning)
            yield
    finally:
        if saved is not missing:
            tarfile.data_filter = saved


def _archive(name: str,
             data: bytes = b'payload',
             typeflag: bytes = tarfile.REGTYPE,
             linkname: str = '') -> io.BytesIO:
    """Build an in-memory tar holding a single member."""
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode='w') as tar:
        info = tarfile.TarInfo(name)
        info.type = typeflag
        if typeflag == tarfile.REGTYPE:
            info.size = len(data)
            tar.addfile(info, io.BytesIO(data))
        else:
            info.linkname = linkname
            tar.addfile(info)
    buf.seek(0)
    return buf


class TestSafeExtract(unittest.TestCase):

    @contextlib.contextmanager
    def _dest(self):
        """Yield (extraction destination, a sibling directory outside it)."""
        with tempfile.TemporaryDirectory() as root:
            dest = os.path.join(root, 'assets')
            outside = os.path.join(root, 'outside')
            os.makedirs(dest)
            os.makedirs(outside)
            yield dest, outside

    def _extract(self, buf: io.BytesIO, dest: str) -> None:
        with tarfile.open(fileobj=buf, mode='r') as tar:
            _safe_extract(tar, dest)

    def test_extracts_ordinary_member(self):
        with self._dest() as (dest, _):
            self._extract(_archive('data.txt', b'hello'), dest)
            with open(os.path.join(dest, 'data.txt'), 'rb') as f:
                self.assertEqual(f.read(), b'hello')

    def test_extracts_nested_member(self):
        with self._dest() as (dest, _):
            self._extract(_archive('pkg/mod.py', b'x = 1'), dest)
            self.assertTrue(os.path.exists(os.path.join(dest, 'pkg', 'mod.py')))

    def test_rejects_parent_traversal(self):
        with self._dest() as (dest, outside):
            with self.assertRaises(_REFUSALS):
                self._extract(_archive('../outside/escaped.txt'), dest)
            self.assertFalse(
                os.path.exists(os.path.join(outside, 'escaped.txt')))

    def test_absolute_member_does_not_escape(self):
        with self._dest() as (dest, outside):
            absolute = os.path.join(outside, 'escaped.txt')
            # The two paths neutralise this differently: the extraction filters
            # strip the leading separator and keep the member inside the
            # destination, while the compatibility path refuses it outright.
            # What matters either way is that nothing lands at the absolute
            # location.
            with contextlib.suppress(*_REFUSALS):
                self._extract(_archive(absolute), dest)
            self.assertFalse(os.path.exists(absolute))

    def test_rejects_symlink_escaping_destination(self):
        with self._dest() as (dest, outside):
            with self.assertRaises(_REFUSALS):
                self._extract(
                    _archive(
                        'link',
                        typeflag=tarfile.SYMTYPE,
                        linkname=os.path.join(outside, 'target')), dest)
            self.assertFalse(os.path.exists(os.path.join(dest, 'link')))

    def test_compatibility_path_extracts_ordinary_member(self):
        with self._dest() as (dest, _):
            with _without_extraction_filters():
                self._extract(_archive('data.txt', b'hello'), dest)
            with open(os.path.join(dest, 'data.txt'), 'rb') as f:
                self.assertEqual(f.read(), b'hello')

    def test_compatibility_path_rejects_parent_traversal(self):
        with self._dest() as (dest, outside):
            with _without_extraction_filters():
                with self.assertRaises(RuntimeError):
                    self._extract(_archive('../outside/escaped.txt'), dest)
            self.assertFalse(
                os.path.exists(os.path.join(outside, 'escaped.txt')))

    def test_compatibility_path_rejects_absolute_member(self):
        with self._dest() as (dest, outside):
            absolute = os.path.join(outside, 'escaped.txt')
            with _without_extraction_filters():
                with self.assertRaises(RuntimeError):
                    self._extract(_archive(absolute), dest)
            self.assertFalse(os.path.exists(absolute))

    def test_compatibility_path_rejects_symlink(self):
        with self._dest() as (dest, outside):
            with _without_extraction_filters():
                with self.assertRaises(RuntimeError):
                    self._extract(
                        _archive(
                            'link',
                            typeflag=tarfile.SYMTYPE,
                            linkname=os.path.join(outside, 'target')), dest)
            self.assertFalse(os.path.exists(os.path.join(dest, 'link')))


class TestSafeExtractSource(unittest.TestCase):

    def test_source_defines_the_helper(self):
        namespace = {}
        exec(safe_extract.get_safe_extract_source(), namespace)  # noqa: S102
        self.assertIn('__kfp_safe_extract', namespace)

    def test_embedded_source_enforces_containment(self):
        namespace = {}
        exec(safe_extract.get_safe_extract_source(), namespace)  # noqa: S102
        embedded = namespace['__kfp_safe_extract']
        with tempfile.TemporaryDirectory() as root:
            dest = os.path.join(root, 'assets')
            os.makedirs(dest)
            with tarfile.open(
                    fileobj=_archive('../escaped.txt'), mode='r') as tar:
                with self.assertRaises(_REFUSALS):
                    embedded(tar, dest)
            self.assertFalse(os.path.exists(os.path.join(root, 'escaped.txt')))


if __name__ == '__main__':
    unittest.main()
