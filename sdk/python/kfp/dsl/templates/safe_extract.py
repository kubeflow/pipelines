"""Template for embedded-archive extraction code generation.

This module contains the extraction helper that gets embedded into generated
component source. Using inspect.getsource() to convert it to source code keeps a
single definition shared by every generator and keeps it directly testable.

Note: The template function imports its dependencies locally, both to avoid
requiring them when this module is imported for code generation and to keep the
generated module namespace free of unprefixed names.
"""

import inspect
import textwrap


def __kfp_safe_extract(tar, path):
    """Extract ``tar`` into ``path`` without letting members escape it."""
    import os as __kfp_os
    import tarfile as __kfp_tarfile

    # tarfile gained the PEP 706 extraction filters in Python 3.12, backported to
    # 3.9.17, 3.10.12 and 3.11.4. Where they exist, 'data' is the policy we want:
    # it refuses absolute paths, traversal outside the destination, links
    # pointing outside it, and special files.
    if hasattr(__kfp_tarfile, 'data_filter'):
        tar.extractall(path=path, filter='data')
        return

    # Older runtimes have no filters at all. Check containment here rather than
    # falling back to an unrestricted extractall.
    __kfp_dest = __kfp_os.path.realpath(path)
    for __kfp_member in tar.getmembers():
        if __kfp_member.issym() or __kfp_member.islnk():
            raise RuntimeError(
                'Refusing to extract link member %r from the embedded archive.'
                % __kfp_member.name)
        if not (__kfp_member.isfile() or __kfp_member.isdir()):
            raise RuntimeError(
                'Refusing to extract special member %r from the embedded archive.'
                % __kfp_member.name)
        __kfp_target = __kfp_os.path.realpath(
            __kfp_os.path.join(__kfp_dest, __kfp_member.name))
        if __kfp_target != __kfp_dest and not __kfp_target.startswith(
                __kfp_dest + __kfp_os.sep):
            raise RuntimeError(
                'Refusing to extract %r outside the embedded asset directory.' %
                __kfp_member.name)
    tar.extractall(path=path)


def get_safe_extract_source() -> str:
    """Return the extraction helper's source code for embedding."""
    return textwrap.dedent(inspect.getsource(__kfp_safe_extract))
