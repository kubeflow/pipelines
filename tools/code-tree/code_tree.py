#!/usr/bin/env python3
"""code_tree.py - Build a knowledge graph of a codebase with call graph and test mapping.

Hybrid Parser + Knowledge Graph + Chunked Context approach:
1. Parse: Extract symbols, calls, and references from every file
2. Index: Link symbols into a knowledge graph with dependency edges
3. Map: Connect tests to source code, detect module boundaries
4. Output: Generate structured artifacts for AI agent consumption

Edge types: contains, imports, inherits, calls, tests, type_of

Usage:
    python code_tree.py [--repo-root .] [--output-dir docs/code-tree]
    python code_tree.py --help

Zero external dependencies required. Optional: tree-sitter for enhanced parsing.
"""

import argparse
import ast as python_ast
import fnmatch
import hashlib
import json
import logging
import os
import re
import sys
from collections import Counter, defaultdict
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional


# ── Constants ─────────────────────────────────────────────────────

LANG_EXTENSIONS = {
    ".py": "python", ".pyw": "python",
    ".go": "go",
    ".js": "javascript", ".jsx": "javascript", ".mjs": "javascript", ".cjs": "javascript",
    ".ts": "typescript", ".tsx": "typescript", ".mts": "typescript",
    ".java": "java",
    ".rs": "rust",
    ".rb": "ruby",
    ".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp", ".hpp": "cpp",
    ".c": "c", ".h": "c",
    ".cs": "csharp",
    ".php": "php",
    ".swift": "swift",
    ".kt": "kotlin", ".kts": "kotlin",
    ".scala": "scala",
    ".proto": "protobuf",
    ".sql": "sql",
    ".sh": "bash", ".bash": "bash",
    ".lua": "lua",
    ".dart": "dart",
    ".ex": "elixir", ".exs": "elixir",
    ".hs": "haskell",
    ".zig": "zig",
    ".r": "r", ".R": "r",
    ".vue": "vue", ".svelte": "svelte",
}

DEFAULT_EXCLUDES = [
    ".git", "node_modules", "__pycache__", ".venv", "venv", "vendor",
    "dist", "build", ".tox", ".mypy_cache", ".pytest_cache", "eggs",
    ".egg-info", ".idea", ".vscode", ".code-tree", "generated",
    ".next", ".nuxt", "coverage", ".cache", ".terraform",
    ".claude",
]

BINARY_EXTENSIONS = {
    ".pyc", ".pyo", ".so", ".o", ".a", ".dylib", ".class", ".jar",
    ".exe", ".dll", ".bin", ".dat", ".db", ".sqlite", ".sqlite3",
    ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".svg",
    ".mp3", ".mp4", ".wav", ".avi", ".mov", ".mkv",
    ".zip", ".tar", ".gz", ".bz2", ".xz", ".rar", ".7z",
    ".woff", ".woff2", ".ttf", ".eot", ".otf",
    ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
}

TEST_MARKERS = {"test_", "_test.", ".test.", ".spec.", "tests/", "__tests__/", "test/"}

BUILD_FILES = {
    "go.mod", "go.sum",
    "package.json",
    "pyproject.toml", "setup.py", "setup.cfg",
    "pom.xml", "build.gradle", "build.gradle.kts",
    "Cargo.toml",
    "Gemfile",
    "composer.json",
    "pubspec.yaml",
    "mix.exs",
}

MAX_FILE_SIZE = 1_048_576  # 1 MB

# Python builtins to exclude from call graph (module-level frozenset for O(1) lookup)
_PYTHON_BUILTINS = frozenset({
    "print", "len", "range", "enumerate", "zip", "map", "filter",
    "sorted", "reversed", "list", "dict", "set", "tuple", "str",
    "int", "float", "bool", "bytes", "type", "isinstance", "issubclass",
    "hasattr", "getattr", "setattr", "delattr", "property",
    "staticmethod", "classmethod", "super", "open", "repr",
    "iter", "next", "any", "all", "min", "max", "sum", "abs",
    "round", "id", "hash", "input", "format", "vars", "dir",
    "callable", "chr", "ord", "hex", "oct", "bin",
    "ValueError", "TypeError", "KeyError", "IndexError",
    "AttributeError", "RuntimeError", "NotImplementedError",
    "StopIteration", "Exception", "OSError", "IOError",
    "FileNotFoundError", "PermissionError", "ImportError",
})

# Keywords to skip in regex call extraction (module-level frozenset)
_REGEX_CALL_KEYWORDS = frozenset({
    "if", "for", "while", "switch", "return", "new", "delete", "throw", "catch",
})


# ── Data Classes ──────────────────────────────────────────────────

@dataclass
class Symbol:
    id: str
    type: str  # file, class, function, method, interface, struct, enum, constant, variable
    name: str
    file: str
    language: str
    line_start: int
    line_end: int
    scope: Optional[str] = None
    params: list[str] = field(default_factory=list)
    bases: list[str] = field(default_factory=list)
    decorators: list[str] = field(default_factory=list)
    docstring: Optional[str] = None
    exported: bool = True
    signature: Optional[str] = None
    is_test: bool = False


@dataclass
class Edge:
    source: str
    target: str
    type: str  # contains, imports, inherits, calls, tests, type_of
    metadata: dict = field(default_factory=dict)


@dataclass
class CallRef:
    """Raw call reference extracted from a function body, before resolution."""
    caller_id: str      # Symbol ID of the calling function/method
    raw_name: str        # Unresolved callee name (e.g. "foo", "self.bar", "pkg.func")
    line: int            # Line number of the call
    file: str            # File containing the call


@dataclass
class ModuleBoundary:
    """A detected build module / package boundary."""
    root_dir: str        # Directory containing the build file
    build_file: str      # Name of the build file (e.g. "package.json")
    name: Optional[str] = None  # Module name if detectable
    language: Optional[str] = None


# ── File Discovery ────────────────────────────────────────────────

def should_exclude(path: str, excludes: list[str]) -> bool:
    parts = Path(path).parts
    for exc in excludes:
        if any(fnmatch.fnmatch(p, exc) for p in parts):
            return True
        if fnmatch.fnmatch(path, exc):
            return True
    return False


def is_binary(file_path: str) -> bool:
    ext = Path(file_path).suffix.lower()
    if ext in BINARY_EXTENSIONS:
        return True
    try:
        with open(file_path, "rb") as f:
            chunk = f.read(8192)
            return b"\x00" in chunk
    except (OSError, IOError):
        return True


def is_test_file(rel_path: str) -> bool:
    """Determine if a file is a test file based on naming conventions."""
    lower_rel = rel_path.lower()
    return any(m in lower_rel for m in TEST_MARKERS)


def file_hash(file_path: str) -> str:
    """Compute SHA-256 hash of a file for change detection."""
    h = hashlib.sha256()
    try:
        with open(file_path, "rb") as f:
            for chunk in iter(lambda: f.read(65536), b""):
                h.update(chunk)
    except OSError:
        return ""
    return h.hexdigest()


def discover_all(
    root: str,
    excludes: list[str],
    include_tests: bool = True,
    max_file_size: int = MAX_FILE_SIZE,
) -> tuple[list[tuple[str, str, str, bool]], list[ModuleBoundary]]:
    """Discover source files and build files in a single os.walk() pass.

    Returns (files, module_boundaries) where files is a list of
    (abs_path, rel_path, language, is_test) tuples.
    """
    files = []
    boundaries = []
    root = os.path.abspath(root)
    exclude_set = set(excludes)

    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [
            d for d in dirnames
            if d not in exclude_set
        ]

        for fname in filenames:
            abs_path = os.path.join(dirpath, fname)

            # Check for build files
            if fname in BUILD_FILES:
                rel_dir = os.path.relpath(dirpath, root)
                if rel_dir == ".":
                    rel_dir = ""
                boundary = ModuleBoundary(root_dir=rel_dir, build_file=fname)
                try:
                    _enrich_module_boundary(abs_path, boundary)
                except (OSError, json.JSONDecodeError, UnicodeDecodeError) as e:
                    logging.debug("Failed to enrich module boundary from %s: %s", abs_path, e)
                boundaries.append(boundary)

            # Check for source files
            ext = Path(fname).suffix.lower()
            lang = LANG_EXTENSIONS.get(ext)
            if not lang:
                continue

            rel_path = os.path.relpath(abs_path, root)
            if should_exclude(rel_path, excludes):
                continue

            test = is_test_file(rel_path)
            if not include_tests and test:
                continue

            try:
                size = os.path.getsize(abs_path)
                if size > max_file_size or size == 0:
                    continue
            except OSError:
                continue

            if is_binary(abs_path):
                continue

            files.append((abs_path, rel_path, lang, test))

    return sorted(files, key=lambda x: x[1]), boundaries


def _enrich_module_boundary(abs_path: str, boundary: ModuleBoundary):
    """Extract module name and language from a build file."""
    fname = boundary.build_file

    if fname == "go.mod":
        boundary.language = "go"
        with open(abs_path, "r", encoding="utf-8") as f:
            for line in f:
                if line.startswith("module "):
                    boundary.name = line.split()[1].strip()
                    break

    elif fname == "package.json":
        boundary.language = "javascript"
        with open(abs_path, "r", encoding="utf-8") as f:
            try:
                data = json.load(f)
                boundary.name = data.get("name", "")
            except json.JSONDecodeError:
                pass

    elif fname in ("pyproject.toml", "setup.cfg"):
        boundary.language = "python"
        with open(abs_path, "r", encoding="utf-8") as f:
            m = _TOML_NAME_RE.search(f.read())
            if m:
                boundary.name = m.group(1)

    elif fname == "Cargo.toml":
        boundary.language = "rust"
        with open(abs_path, "r", encoding="utf-8") as f:
            m = _TOML_NAME_RE.search(f.read())
            if m:
                boundary.name = m.group(1)

    elif fname in ("pom.xml",):
        boundary.language = "java"
        with open(abs_path, "r", encoding="utf-8") as f:
            content = f.read()
            m = re.search(r'<artifactId>([^<]+)</artifactId>', content)
            if m:
                boundary.name = m.group(1)

    elif fname in ("build.gradle", "build.gradle.kts"):
        boundary.language = "java"


def read_file(path: str) -> Optional[str]:
    for encoding in ("utf-8", "latin-1"):
        try:
            with open(path, "r", encoding=encoding) as f:
                return f.read()
        except (UnicodeDecodeError, OSError):
            continue
    return None


# ── Python Parser (ast module) ────────────────────────────────────

def parse_python(
    content: str, rel_path: str, is_test: bool,
) -> tuple[list[Symbol], list[Edge], list[CallRef]]:
    """Parse Python using the ast module with call graph extraction."""
    symbols = []
    edges = []
    call_refs = []
    file_id = f"file:{rel_path}"

    try:
        tree = python_ast.parse(content, filename=rel_path)
    except SyntaxError:
        return symbols, edges, call_refs

    # Extract imports
    for node in python_ast.walk(tree):
        if isinstance(node, python_ast.Import):
            for alias in node.names:
                edges.append(Edge(file_id, f"module:{alias.name}", "imports"))
        elif isinstance(node, python_ast.ImportFrom):
            if node.module:
                level = node.level or 0
                module_ref = node.module
                if level > 0:
                    # Relative import: compute target module
                    parts = Path(rel_path).parts
                    if len(parts) > level:
                        base = ".".join(parts[:len(parts) - level])
                        module_ref = f"{base}.{node.module}" if node.module else base
                edges.append(Edge(file_id, f"module:{module_ref}", "imports"))
                for alias in (node.names or []):
                    if alias.name != "*":
                        edges.append(Edge(
                            file_id,
                            f"symbol:{module_ref}.{alias.name}",
                            "imports",
                            {"from_import": True},
                        ))

    # Walk top-level nodes for symbols
    _walk_python_body(
        tree, rel_path, file_id, None, symbols, edges, call_refs, is_test,
    )

    return symbols, edges, call_refs


def _walk_python_body(
    node, rel_path: str, parent_id: str, scope: Optional[str],
    symbols: list, edges: list, call_refs: list, is_test: bool,
):
    """Recursively walk Python AST nodes to extract symbols and calls."""
    for child in python_ast.iter_child_nodes(node):
        if isinstance(child, python_ast.ClassDef):
            _extract_python_class(
                child, rel_path, parent_id, scope, symbols, edges, call_refs, is_test,
            )
        elif isinstance(child, (python_ast.FunctionDef, python_ast.AsyncFunctionDef)):
            _extract_python_function(
                child, rel_path, parent_id, scope, symbols, edges, call_refs, is_test,
            )
        elif isinstance(child, python_ast.Assign):
            # Module-level constants (UPPER_CASE)
            if scope is None:  # Only at module level
                for target in child.targets:
                    if isinstance(target, python_ast.Name) and target.id.isupper():
                        sym_id = f"constant:{rel_path}:{target.id}"
                        symbols.append(Symbol(
                            id=sym_id, type="constant", name=target.id,
                            file=rel_path, language="python",
                            line_start=child.lineno,
                            line_end=child.end_lineno or child.lineno,
                            exported=not target.id.startswith("_"),
                            is_test=is_test,
                        ))
                        edges.append(Edge(parent_id, sym_id, "contains"))


def _extract_python_class(
    node: python_ast.ClassDef, rel_path: str, parent_id: str,
    outer_scope: Optional[str],
    symbols: list, edges: list, call_refs: list, is_test: bool,
):
    """Extract a Python class, including nested classes and methods."""
    qualified = f"{outer_scope}.{node.name}" if outer_scope else node.name
    sym_id = f"class:{rel_path}:{qualified}"

    bases = []
    for base in node.bases:
        try:
            bases.append(python_ast.unparse(base))
        except (ValueError, TypeError):
            if isinstance(base, python_ast.Name):
                bases.append(base.id)

    decorators = []
    for dec in node.decorator_list:
        try:
            decorators.append(python_ast.unparse(dec))
        except (ValueError, TypeError):
            pass

    docstring = python_ast.get_docstring(node)
    symbols.append(Symbol(
        id=sym_id, type="class", name=node.name,
        file=rel_path, language="python",
        line_start=node.lineno,
        line_end=node.end_lineno or node.lineno,
        scope=outer_scope,
        bases=bases, decorators=decorators,
        docstring=docstring[:200] if docstring else None,
        exported=not node.name.startswith("_"),
        is_test=is_test,
    ))
    edges.append(Edge(parent_id, sym_id, "contains"))

    for base_name in bases:
        edges.append(Edge(sym_id, f"class:?:{base_name}", "inherits", {"unresolved": True}))

    # Recurse into class body — handles nested classes, methods, inner functions
    _walk_python_body(
        node, rel_path, sym_id, qualified, symbols, edges, call_refs, is_test,
    )


def _extract_python_function(
    node, rel_path: str, parent_id: str, scope: Optional[str],
    symbols: list, edges: list, call_refs: list, is_test: bool,
):
    """Extract a Python function/method with call graph analysis."""
    qualified = f"{scope}.{node.name}" if scope else node.name

    # Determine if this is a method (parent is a class)
    is_method = scope is not None and parent_id.startswith("class:")
    sym_type = "method" if is_method else "function"

    sym_id = f"{sym_type}:{rel_path}:{qualified}"

    params = [a.arg for a in node.args.args if a.arg != "self"]
    docstring = python_ast.get_docstring(node)
    decorators = []
    for dec in node.decorator_list:
        try:
            decorators.append(python_ast.unparse(dec))
        except (ValueError, TypeError):
            pass

    # Build signature
    sig_parts = []
    for a in node.args.args:
        part = a.arg
        if a.annotation:
            try:
                part += f": {python_ast.unparse(a.annotation)}"
            except (ValueError, TypeError):
                pass
        sig_parts.append(part)
    signature = f"({', '.join(sig_parts)})"
    if hasattr(node, 'returns') and node.returns:
        try:
            signature += f" -> {python_ast.unparse(node.returns)}"
        except (ValueError, TypeError):
            pass

    symbols.append(Symbol(
        id=sym_id, type=sym_type, name=node.name,
        file=rel_path, language="python",
        line_start=node.lineno,
        line_end=node.end_lineno or node.lineno,
        scope=scope, params=params, decorators=decorators,
        docstring=docstring[:200] if docstring else None,
        exported=not node.name.startswith("_"),
        signature=signature,
        is_test=is_test or node.name.startswith("test_"),
    ))
    edges.append(Edge(parent_id, sym_id, "contains"))

    # Extract calls from function body
    _extract_python_calls(node, sym_id, rel_path, call_refs)

    # Recurse into nested functions/classes inside this function
    for child in python_ast.iter_child_nodes(node):
        if isinstance(child, python_ast.ClassDef):
            _extract_python_class(
                child, rel_path, sym_id, qualified, symbols, edges, call_refs, is_test,
            )
        elif isinstance(child, (python_ast.FunctionDef, python_ast.AsyncFunctionDef)):
            # Nested function (closure)
            _extract_python_function(
                child, rel_path, sym_id, qualified, symbols, edges, call_refs, is_test,
            )


def _extract_python_calls(
    func_node, caller_id: str, rel_path: str,
    call_refs: list,
):
    """Walk a Python function body to extract call references.

    Uses a manual stack instead of ast.walk() to skip nested
    FunctionDef/AsyncFunctionDef/ClassDef nodes (those are handled
    separately and would cause double-counting).
    """
    stack = list(python_ast.iter_child_nodes(func_node))
    while stack:
        node = stack.pop()

        # Skip nested function/class definitions — they have their own caller_id
        if isinstance(node, (python_ast.FunctionDef, python_ast.AsyncFunctionDef, python_ast.ClassDef)):
            continue

        if isinstance(node, python_ast.Call):
            raw_name = None
            line = getattr(node, 'lineno', 0)

            if isinstance(node.func, python_ast.Name):
                raw_name = node.func.id
            elif isinstance(node.func, python_ast.Attribute):
                try:
                    raw_name = python_ast.unparse(node.func)
                except (ValueError, TypeError):
                    raw_name = node.func.attr

            if raw_name and raw_name not in _PYTHON_BUILTINS:
                call_refs.append(CallRef(
                    caller_id=caller_id,
                    raw_name=raw_name,
                    line=line,
                    file=rel_path,
                ))

        # Continue walking children
        stack.extend(python_ast.iter_child_nodes(node))


# ── Regex Parsers ─────────────────────────────────────────────────

def _compile_patterns(raw_patterns):
    """Pre-compile a list of (regex_str, sym_type, groups) into (compiled_re, sym_type, groups)."""
    return [(re.compile(pat), sym_type, groups) for pat, sym_type, groups in raw_patterns]

GO_PATTERNS = _compile_patterns([
    (r"^package\s+(\w+)", "package", {"name": 1}),
    (r'^func\s+(\w+)\s*\(([^)]*)\)', "function", {"name": 1, "params": 2}),
    (r'^func\s+\(\w+\s+\*?(\w+)\)\s+(\w+)\s*\(([^)]*)\)', "method", {"scope": 1, "name": 2, "params": 3}),
    (r'^type\s+(\w+)\s+struct\s*\{', "struct", {"name": 1}),
    (r'^type\s+(\w+)\s+interface\s*\{', "interface", {"name": 1}),
    (r'^\s*"([^"]+)"', "_import", {"name": 1}),
])

GO_IMPORT_BLOCK = re.compile(r'^import\s*\(\s*\n(.*?)\n\s*\)', re.MULTILINE | re.DOTALL)
GO_IMPORT_SINGLE = re.compile(r'^import\s+"([^"]+)"', re.MULTILINE)

JS_PATTERNS = _compile_patterns([
    (r"^(?:export\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?", "class", {"name": 1, "bases": 2}),
    (r"^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)", "function", {"name": 1, "params": 2}),
    (r"^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:\([^)]*\)|[^=])\s*=>", "function", {"name": 1}),
    (r'^import\s+.*?\s+from\s+[\'"]([^\'"]+)[\'"]', "_import", {"name": 1}),
    (r'^import\s+[\'"]([^\'"]+)[\'"]', "_import", {"name": 1}),
    (r"require\(\s*['\"]([^'\"]+)['\"]\s*\)", "_import", {"name": 1}),
    (r"^(?:export\s+)?interface\s+(\w+)", "interface", {"name": 1}),
    (r"^(?:export\s+)?type\s+(\w+)\s*=", "interface", {"name": 1}),
    (r"^(?:export\s+)?enum\s+(\w+)", "enum", {"name": 1}),
])

JAVA_PATTERNS = _compile_patterns([
    (r"^package\s+([\w.]+);", "package", {"name": 1}),
    (r"^import\s+([\w.]+);", "_import", {"name": 1}),
    (r"(?:public|private|protected)?\s*(?:abstract\s+)?(?:static\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?", "class", {"name": 1, "bases": 2}),
    (r"(?:public|private|protected)?\s*interface\s+(\w+)", "interface", {"name": 1}),
    (r"(?:public|private|protected)?\s*enum\s+(\w+)", "enum", {"name": 1}),
    (r"(?:public|private|protected)\s+(?:static\s+)?(?:abstract\s+)?(?:synchronized\s+)?(?:final\s+)?\w+(?:<[^>]+>)?\s+(\w+)\s*\(([^)]*)\)", "method", {"name": 1, "params": 2}),
])

RUST_PATTERNS = _compile_patterns([
    (r"^pub\s+fn\s+(\w+)\s*(?:<[^>]*>)?\s*\(([^)]*)\)", "function", {"name": 1, "params": 2}),
    (r"^fn\s+(\w+)\s*(?:<[^>]*>)?\s*\(([^)]*)\)", "function", {"name": 1, "params": 2}),
    (r"^pub\s+struct\s+(\w+)", "struct", {"name": 1}),
    (r"^struct\s+(\w+)", "struct", {"name": 1}),
    (r"^pub\s+enum\s+(\w+)", "enum", {"name": 1}),
    (r"^enum\s+(\w+)", "enum", {"name": 1}),
    (r"^pub\s+trait\s+(\w+)", "interface", {"name": 1}),
    (r"^trait\s+(\w+)", "interface", {"name": 1}),
    (r"^impl(?:<[^>]*>)?\s+(\w+)", "struct", {"name": 1}),
    (r'^use\s+([\w:]+)', "_import", {"name": 1}),
    (r'^mod\s+(\w+);', "_import", {"name": 1}),
])

PROTO_PATTERNS = _compile_patterns([
    (r'^message\s+(\w+)\s*\{', "class", {"name": 1}),
    (r'^service\s+(\w+)\s*\{', "interface", {"name": 1}),
    (r'^\s*rpc\s+(\w+)\s*\((\w+)\)\s*returns\s*\((\w+)\)', "method", {"name": 1, "params": 2}),
    (r'^enum\s+(\w+)\s*\{', "enum", {"name": 1}),
    (r'^import\s+"([^"]+)"', "_import", {"name": 1}),
])

CSHARP_PATTERNS = _compile_patterns([
    (r"(?:public|private|protected|internal)?\s*(?:abstract\s+)?(?:static\s+)?class\s+(\w+)(?:\s*:\s*(\w+))?", "class", {"name": 1, "bases": 2}),
    (r"(?:public|private|protected|internal)?\s*interface\s+(\w+)", "interface", {"name": 1}),
    (r"(?:public|private|protected|internal)?\s*enum\s+(\w+)", "enum", {"name": 1}),
    (r"(?:public|private|protected|internal)\s+(?:static\s+)?(?:async\s+)?(?:virtual\s+)?(?:override\s+)?\w+(?:<[^>]+>)?\s+(\w+)\s*\(([^)]*)\)", "method", {"name": 1, "params": 2}),
    (r'^using\s+([\w.]+);', "_import", {"name": 1}),
])

LANG_PATTERNS = {
    "go": GO_PATTERNS,
    "javascript": JS_PATTERNS,
    "typescript": JS_PATTERNS,
    "java": JAVA_PATTERNS,
    "rust": RUST_PATTERNS,
    "protobuf": PROTO_PATTERNS,
    "csharp": CSHARP_PATTERNS,
}

# Regex pattern for extracting function calls from source code
CALL_PATTERN = re.compile(r'\b([a-zA-Z_]\w*(?:\.[a-zA-Z_]\w*)*)\s*\(')

# Shared TOML name extraction pattern (used for pyproject.toml and Cargo.toml)
_TOML_NAME_RE = re.compile(r'name\s*=\s*"([^"]+)"')


def _extract_go_import_edges(content: str, file_id: str) -> list[Edge]:
    """Extract Go import edges from content (shared by regex and tree-sitter paths)."""
    edges = []
    for m in GO_IMPORT_BLOCK.finditer(content):
        block = m.group(1)
        for line in block.strip().split("\n"):
            line = line.strip().strip('"')
            if line and not line.startswith("//"):
                parts = line.split()
                imp = parts[-1].strip('"') if parts else line
                edges.append(Edge(file_id, f"module:{imp}", "imports"))
    for m in GO_IMPORT_SINGLE.finditer(content):
        edges.append(Edge(file_id, f"module:{m.group(1)}", "imports"))
    return edges


def parse_with_regex(
    content: str, rel_path: str, language: str, is_test: bool,
) -> tuple[list[Symbol], list[Edge], list[CallRef]]:
    """Parse a source file using regex patterns with call extraction."""
    symbols = []
    edges = []
    call_refs = []
    file_id = f"file:{rel_path}"
    patterns = LANG_PATTERNS.get(language, [])
    if not patterns:
        return symbols, edges, call_refs

    lines = content.split("\n")

    # Handle Go import blocks
    if language == "go":
        edges.extend(_extract_go_import_edges(content, file_id))

    current_class = None
    brace_depth = 0
    symbol_ranges = []  # Track (line_start, sym_id, sym_type) for call extraction

    for lineno, line in enumerate(lines, 1):
        stripped = line.rstrip()

        brace_depth += stripped.count("{") - stripped.count("}")
        if brace_depth <= 0:
            current_class = None

        for compiled_re, sym_type, groups in patterns:
            m = compiled_re.search(stripped)
            if not m:
                continue

            name = m.group(groups["name"]) if "name" in groups else None
            if not name:
                continue

            if sym_type == "_import":
                if language != "go":
                    edges.append(Edge(file_id, f"module:{name}", "imports"))
                continue

            if sym_type == "package":
                continue

            params_str = m.group(groups["params"]) if "params" in groups and groups["params"] <= len(m.groups()) else None
            params = [p.strip().split()[-1].split(":")[0] for p in params_str.split(",") if p.strip()] if params_str else []

            bases_match = m.group(groups["bases"]) if "bases" in groups and groups["bases"] <= len(m.groups()) and m.group(groups["bases"]) else None
            bases = [bases_match] if bases_match else []

            scope_match = m.group(groups["scope"]) if "scope" in groups and groups["scope"] <= len(m.groups()) and m.group(groups["scope"]) else None
            scope = scope_match or (current_class if sym_type == "method" else None)

            # Determine end line (use len(lines) + 1 so the last line is reachable)
            end_line = lineno
            for j in range(lineno, min(lineno + 500, len(lines) + 1)):
                if j > lineno and j <= len(lines) and lines[j - 1].strip() and not lines[j - 1].startswith((" ", "\t")) and brace_depth <= 1:
                    end_line = j - 1
                    break
            else:
                end_line = min(lineno + 50, len(lines))

            if scope:
                sym_id = f"{sym_type}:{rel_path}:{scope}.{name}"
            else:
                sym_id = f"{sym_type}:{rel_path}:{name}"

            exported = not name.startswith("_")
            if language == "go":
                exported = name[0].isupper() if name else True

            symbols.append(Symbol(
                id=sym_id, type=sym_type, name=name,
                file=rel_path, language=language,
                line_start=lineno, line_end=end_line,
                scope=scope, params=params, bases=bases,
                exported=exported,
                is_test=is_test,
            ))

            if sym_type in ("class", "struct", "interface", "enum"):
                edges.append(Edge(file_id, sym_id, "contains"))
                current_class = name
                for b in bases:
                    edges.append(Edge(sym_id, f"class:?:{b}", "inherits", {"unresolved": True}))
            elif sym_type == "method" and scope:
                parent_id = f"class:{rel_path}:{scope}"
                edges.append(Edge(parent_id, sym_id, "contains"))
            elif sym_type == "function":
                edges.append(Edge(file_id, sym_id, "contains"))

            symbol_ranges.append((lineno, end_line, sym_id))
            break

    # Extract calls from function/method bodies using regex
    _extract_regex_calls(lines, symbol_ranges, rel_path, call_refs)

    return symbols, edges, call_refs


def _extract_regex_calls(
    lines: list[str], symbol_ranges: list, rel_path: str, call_refs: list,
):
    """Extract function calls from source code using regex (fallback)."""
    for start, end, sym_id in symbol_ranges:
        if not (sym_id.startswith("function:") or sym_id.startswith("method:")):
            continue

        for lineno in range(start, min(end + 1, len(lines) + 1)):
            line = lines[lineno - 1] if lineno <= len(lines) else ""
            # Skip comments and strings (rough heuristic)
            stripped = line.strip()
            if stripped.startswith("//") or stripped.startswith("#") or stripped.startswith("*"):
                continue

            for m in CALL_PATTERN.finditer(line):
                raw_name = m.group(1)
                if raw_name in _REGEX_CALL_KEYWORDS:
                    continue
                call_refs.append(CallRef(
                    caller_id=sym_id,
                    raw_name=raw_name,
                    line=lineno,
                    file=rel_path,
                ))


def _extract_regex_imports(
    content: str, rel_path: str, language: str,
) -> list[Edge]:
    """Extract only import edges using regex — lightweight fallback for tree-sitter."""
    edges = []
    file_id = f"file:{rel_path}"
    patterns = LANG_PATTERNS.get(language, [])

    # Handle Go import blocks
    if language == "go":
        return _extract_go_import_edges(content, file_id)

    # For other languages, scan only import patterns
    for line in content.split("\n"):
        stripped = line.rstrip()
        for compiled_re, sym_type, groups in patterns:
            if sym_type != "_import":
                continue
            m = compiled_re.search(stripped)
            if m:
                name = m.group(groups["name"]) if "name" in groups else None
                if name:
                    edges.append(Edge(file_id, f"module:{name}", "imports"))

    return edges


# ── Tree-sitter Integration ──────────────────────────────────────

def try_load_tree_sitter() -> dict:
    """Try to load tree-sitter with available language grammars."""
    available = {}

    try:
        from tree_sitter import Language, Parser
    except ImportError:
        return available

    grammar_packages = {
        "python": "tree_sitter_python",
        "go": "tree_sitter_go",
        "javascript": "tree_sitter_javascript",
        "typescript": "tree_sitter_typescript",
        "java": "tree_sitter_java",
        "rust": "tree_sitter_rust",
        "ruby": "tree_sitter_ruby",
        "c": "tree_sitter_c",
        "cpp": "tree_sitter_cpp",
    }

    for lang, pkg in grammar_packages.items():
        try:
            mod = __import__(pkg)
            language_func = getattr(mod, "language", None)
            if language_func:
                lang_obj = Language(language_func())
                parser = Parser(lang_obj)
                available[lang] = (parser, lang_obj)
        except (ImportError, AttributeError, OSError) as e:
            logging.debug("tree-sitter grammar %s unavailable: %s", pkg, e)

    if not available:
        try:
            from tree_sitter_languages import get_parser, get_language
            for lang in ["python", "go", "javascript", "typescript", "java", "rust", "ruby", "c", "cpp"]:
                try:
                    available[lang] = (get_parser(lang), get_language(lang))
                except (ImportError, AttributeError, OSError) as e:
                    logging.debug("tree-sitter-languages %s unavailable: %s", lang, e)
        except ImportError:
            pass

    return available


# Node types that represent symbol declarations per language
TS_SYMBOL_TYPES = {
    "python": {
        "class_definition": "class",
        "function_definition": "function",
    },
    "go": {
        "function_declaration": "function",
        "method_declaration": "method",
        "type_declaration": None,  # need sub-check for struct vs interface
    },
    "javascript": {
        "class_declaration": "class",
        "function_declaration": "function",
        "arrow_function": "function",
        "method_definition": "method",
        "interface_declaration": "interface",
    },
    "typescript": {
        "class_declaration": "class",
        "function_declaration": "function",
        "arrow_function": "function",
        "method_definition": "method",
        "interface_declaration": "interface",
        "type_alias_declaration": "interface",
        "enum_declaration": "enum",
    },
    "java": {
        "class_declaration": "class",
        "interface_declaration": "interface",
        "enum_declaration": "enum",
        "method_declaration": "method",
        "constructor_declaration": "method",
    },
    "rust": {
        "function_item": "function",
        "struct_item": "struct",
        "enum_item": "enum",
        "trait_item": "interface",
        "impl_item": None,  # scope provider, not a symbol itself
    },
}

# Node types that represent function calls per language
TS_CALL_TYPES = {
    "python": ["call"],
    "go": ["call_expression"],
    "javascript": ["call_expression", "new_expression"],
    "typescript": ["call_expression", "new_expression"],
    "java": ["method_invocation", "object_creation_expression"],
    "rust": ["call_expression", "macro_invocation"],
}

def parse_with_tree_sitter(
    content: str, rel_path: str, language: str, parser, lang_obj, is_test: bool,
) -> tuple[list[Symbol], list[Edge], list[CallRef]]:
    """Parse a file using tree-sitter with full call graph extraction."""
    symbols = []
    edges = []
    call_refs = []
    file_id = f"file:{rel_path}"

    content_bytes = content.encode("utf-8")
    try:
        tree = parser.parse(content_bytes)
    except (ValueError, TypeError, RuntimeError) as e:
        logging.debug("tree-sitter parse failed for %s: %s", rel_path, e)
        return symbols, edges, call_refs
    symbol_types = TS_SYMBOL_TYPES.get(language, {})
    call_types = set(TS_CALL_TYPES.get(language, []))

    def get_node_text(node):
        return content_bytes[node.start_byte:node.end_byte].decode("utf-8", errors="replace")

    def find_name(node):
        """Extract the name identifier from a declaration node."""
        # Try field name first
        name_node = node.child_by_field_name("name")
        if name_node:
            return get_node_text(name_node)

        # Try first identifier child
        for child in node.children:
            if child.type in ("identifier", "name", "type_identifier", "property_identifier"):
                return get_node_text(child)
        return None

    def find_params(node):
        """Extract parameter names from a function/method declaration."""
        params = []
        param_node = node.child_by_field_name("parameters") or node.child_by_field_name("formal_parameters")
        if not param_node:
            for child in node.children:
                if child.type in ("parameters", "formal_parameters", "parameter_list"):
                    param_node = child
                    break

        if param_node:
            for child in param_node.children:
                if child.type in ("identifier", "name"):
                    text = get_node_text(child)
                    if text != "self" and text != "cls":
                        params.append(text)
                elif child.type in ("parameter", "formal_parameter", "typed_parameter",
                                     "default_parameter", "typed_default_parameter"):
                    name_n = child.child_by_field_name("name")
                    if not name_n:
                        for sub in child.children:
                            if sub.type == "identifier":
                                name_n = sub
                                break
                    if name_n:
                        text = get_node_text(name_n)
                        if text != "self" and text != "cls":
                            params.append(text)
        return params

    def find_bases(node, language):
        """Extract base classes/interfaces from a class declaration."""
        bases = []
        # Python: argument_list in class_definition
        superclass_node = node.child_by_field_name("superclasses")
        if superclass_node:
            for child in superclass_node.children:
                if child.type in ("identifier", "attribute"):
                    bases.append(get_node_text(child))
            return bases

        # Java/JS/TS: superclass field
        for field_name in ("superclass", "super_class"):
            sc = node.child_by_field_name(field_name)
            if sc:
                bases.append(get_node_text(sc))

        # Look for extends/implements clauses
        for child in node.children:
            if child.type in ("superclass", "class_heritage"):
                for sub in child.children:
                    if sub.type in ("identifier", "type_identifier"):
                        bases.append(get_node_text(sub))
        return bases

    def extract_calls_from_body(body_node, caller_id):
        """Walk a function body to extract call references."""
        if body_node is None:
            return

        stack = [body_node]
        while stack:
            node = stack.pop()

            if node.type in call_types:
                func_node = node.child_by_field_name("function")
                if not func_node:
                    # Try first child for some languages
                    if node.children:
                        func_node = node.children[0]

                if func_node:
                    raw_name = get_node_text(func_node)
                    # Clean up the name
                    raw_name = raw_name.strip()
                    if raw_name and len(raw_name) < 200:  # sanity limit
                        call_refs.append(CallRef(
                            caller_id=caller_id,
                            raw_name=raw_name,
                            line=node.start_point[0] + 1,
                            file=rel_path,
                        ))

            # Don't recurse into nested function/class declarations for calls
            if node.type not in symbol_types and node != body_node:
                for child in node.children:
                    stack.append(child)
            elif node == body_node:
                for child in node.children:
                    stack.append(child)

    def visit(node, scope=None, parent_id=None):
        """Recursively visit tree-sitter AST nodes."""
        if parent_id is None:
            parent_id = file_id

        node_type = node.type

        # Handle scope-only nodes (e.g., impl_item in Rust) that provide
        # scope context for their children but are not symbols themselves
        if node_type in symbol_types and symbol_types[node_type] is None:
            impl_name = find_name(node)
            if impl_name:
                for child in node.children:
                    visit(child, scope=impl_name, parent_id=parent_id)
            else:
                for child in node.children:
                    visit(child, scope=scope, parent_id=parent_id)
            return

        sym_type = symbol_types.get(node_type)

        if sym_type is not None:
            name = find_name(node)
            if name:
                qualified = f"{scope}.{name}" if scope else name

                # Determine actual sym_type for special cases
                actual_type = sym_type
                if language == "go" and node_type == "type_declaration":
                    # Check if struct or interface
                    for child in node.children:
                        if child.type == "type_spec":
                            for sub in child.children:
                                if sub.type == "struct_type":
                                    actual_type = "struct"
                                elif sub.type == "interface_type":
                                    actual_type = "interface"
                            name_node = child.child_by_field_name("name")
                            if name_node:
                                name = get_node_text(name_node)
                                qualified = f"{scope}.{name}" if scope else name

                # Determine if method
                if actual_type == "function" and scope and parent_id.startswith("class:"):
                    actual_type = "method"

                sym_id = f"{actual_type}:{rel_path}:{qualified}"

                exported = not name.startswith("_")
                if language == "go":
                    exported = name[0].isupper() if name else True

                params = find_params(node)
                bases = find_bases(node, language) if actual_type in ("class", "struct") else []

                symbols.append(Symbol(
                    id=sym_id, type=actual_type, name=name,
                    file=rel_path, language=language,
                    line_start=node.start_point[0] + 1,
                    line_end=node.end_point[0] + 1,
                    scope=scope, params=params, bases=bases,
                    exported=exported,
                    is_test=is_test,
                ))

                # Create containment edges
                if actual_type in ("class", "struct", "interface", "enum"):
                    edges.append(Edge(parent_id, sym_id, "contains"))
                    for b in bases:
                        edges.append(Edge(sym_id, f"class:?:{b}", "inherits", {"unresolved": True}))
                    # Recurse into class body with scope
                    for child in node.children:
                        visit(child, scope=name, parent_id=sym_id)
                    return
                elif actual_type == "method" and scope:
                    scope_id = f"class:{rel_path}:{scope}"
                    edges.append(Edge(scope_id, sym_id, "contains"))
                else:
                    edges.append(Edge(parent_id, sym_id, "contains"))

                # Extract calls from function/method body
                body_node = node.child_by_field_name("body") or node.child_by_field_name("block")
                if body_node is None:
                    # Some languages use the last block child
                    for child in reversed(node.children):
                        if child.type in ("block", "statement_block", "compound_statement"):
                            body_node = child
                            break
                extract_calls_from_body(body_node, sym_id)
                return  # Don't recurse into function body for more symbols (nested handled above)

        # Not a symbol node; recurse
        for child in node.children:
            visit(child, scope, parent_id)

    visit(tree.root_node)

    # Extract imports using regex (more reliable for multi-language consistency)
    edges.extend(_extract_regex_imports(content, rel_path, language))

    return symbols, edges, call_refs


# ── Parser Dispatcher ─────────────────────────────────────────────

def parse_file(
    abs_path: str, rel_path: str, language: str,
    ts_parsers: dict, is_test: bool,
) -> tuple[list[Symbol], list[Edge], list[CallRef], int]:
    """Parse a single file, choosing the best available parser.

    Returns (symbols, edges, call_refs, line_count).
    """
    content = read_file(abs_path)
    if content is None:
        return [], [], [], 0

    line_count = content.count("\n") + (1 if content and not content.endswith("\n") else 0)

    # Python: always use ast module (most accurate)
    if language == "python":
        symbols, edges, calls = parse_python(content, rel_path, is_test)
        return symbols, edges, calls, line_count

    # Tree-sitter if available
    if language in ts_parsers:
        parser, lang_obj = ts_parsers[language]
        symbols, edges, calls = parse_with_tree_sitter(
            content, rel_path, language, parser, lang_obj, is_test,
        )
        if symbols or edges:
            return symbols, edges, calls, line_count

    # Fallback: regex
    symbols, edges, calls = parse_with_regex(content, rel_path, language, is_test)
    return symbols, edges, calls, line_count


# ── Call Resolution ───────────────────────────────────────────────

def resolve_calls(
    call_refs: list[CallRef],
    symbols: list[Symbol],
    edges: list[Edge],
    file_map: dict,
) -> list[Edge]:
    """Resolve raw call references to known symbols, creating calls edges."""
    call_edges = []

    # Build lookup indexes
    # name -> list of symbol IDs
    name_to_symbols = defaultdict(list)
    for sym in symbols:
        name_to_symbols[sym.name].append(sym)
        if sym.scope:
            name_to_symbols[f"{sym.scope}.{sym.name}"].append(sym)

    # file -> {imported_name -> module_path}
    file_imports = defaultdict(dict)
    for edge in edges:
        if edge.type == "imports" and edge.source.startswith("file:"):
            src_file = edge.source[5:]
            if edge.target.startswith("symbol:"):
                # from X import Y -> local name Y maps to symbol
                symbol_path = edge.target[7:]
                parts = symbol_path.rsplit(".", 1)
                if len(parts) == 2:
                    local_name = parts[1]
                    file_imports[src_file][local_name] = symbol_path
            elif edge.target.startswith("module:"):
                mod_name = edge.target[7:]
                # import X -> local name X
                local_name = mod_name.rsplit(".", 1)[-1]
                file_imports[src_file][local_name] = mod_name

    # file -> list of symbols defined in that file
    file_symbols = defaultdict(list)
    for sym in symbols:
        file_symbols[sym.file].append(sym)

    # Build a set of all symbol IDs for O(1) existence checks
    symbol_ids = {s.id for s in symbols}

    # Deduplicate calls: (caller_id, resolved_target_id) pairs
    seen_calls = set()

    for ref in call_refs:
        raw = ref.raw_name
        caller_id = ref.caller_id

        resolved = None

        # 1. self.method() -> look up method in the same class
        if raw.startswith("self.") or raw.startswith("this."):
            method_name = raw.split(".", 1)[1]
            # Find the class scope from caller_id
            # caller_id format: method:file:Class.method_name
            parts = caller_id.split(":")
            if len(parts) >= 3:
                scope_parts = parts[2].rsplit(".", 1)
                if len(scope_parts) == 2:
                    class_name = scope_parts[0]
                    target_id = f"method:{ref.file}:{class_name}.{method_name}"
                    if target_id in symbol_ids:
                        resolved = target_id

        # 2. Direct name -> look up in same file, then imports
        if not resolved and "." not in raw:
            # Same file first
            for sym in file_symbols.get(ref.file, []):
                if sym.name == raw and sym.id != caller_id:
                    resolved = sym.id
                    break

            # Check imports
            if not resolved:
                imported_path = file_imports.get(ref.file, {}).get(raw)
                if imported_path:
                    # Find the actual symbol
                    parts = imported_path.rsplit(".", 1)
                    if len(parts) == 2:
                        sym_name = parts[1]
                        for sym in name_to_symbols.get(sym_name, []):
                            resolved = sym.id
                            break

        # 3. Qualified name: module.func or Class.method
        if not resolved and "." in raw:
            parts = raw.split(".")
            # Try as imported_module.symbol
            if len(parts) == 2:
                obj, attr = parts
                # Check if obj is an imported module
                imported_path = file_imports.get(ref.file, {}).get(obj)
                if imported_path:
                    for sym in name_to_symbols.get(attr, []):
                        resolved = sym.id
                        break

                # Check if obj is a class in the same file -> static method or class ref
                if not resolved:
                    for sym in file_symbols.get(ref.file, []):
                        if sym.name == obj and sym.type in ("class", "struct"):
                            target_id = f"method:{ref.file}:{obj}.{attr}"
                            if target_id in symbol_ids:
                                resolved = target_id
                                break

        # 4. Last resort: match by name across all symbols
        if not resolved:
            simple_name = raw.split(".")[-1]
            candidates = name_to_symbols.get(simple_name, [])
            if len(candidates) == 1 and candidates[0].id != caller_id:
                resolved = candidates[0].id

        if resolved:
            key = (caller_id, resolved)
            if key not in seen_calls and caller_id != resolved:
                seen_calls.add(key)
                call_edges.append(Edge(
                    caller_id, resolved, "calls",
                    {"line": ref.line},
                ))

    return call_edges


# ── Test-to-Code Mapping ─────────────────────────────────────────

def detect_test_relationships(
    symbols: list[Symbol],
    edges: list[Edge],
    files: list[tuple[str, str, str, bool]],
    file_map: dict,
) -> list[Edge]:
    """Map test files/functions to the source code they test."""
    test_edges = []
    seen = set()

    test_files = [(abs_p, rel_p, lang) for abs_p, rel_p, lang, is_test in files if is_test]
    source_files = {rel_p for _, rel_p, _, is_test in files if not is_test}

    # Build indexes (bare name and qualified scope.name for method-level matching)
    symbol_by_name = defaultdict(list)
    for sym in symbols:
        if not sym.is_test:
            symbol_by_name[sym.name].append(sym)
            if sym.scope:
                symbol_by_name[f"{sym.scope}.{sym.name}"].append(sym)

    # Strategy 1: Path mirroring
    for _, test_rel, lang in test_files:
        source_match = _match_test_to_source_path(test_rel, source_files)
        if source_match:
            key = (f"file:{test_rel}", f"file:{source_match}")
            if key not in seen:
                seen.add(key)
                test_edges.append(Edge(
                    f"file:{test_rel}", f"file:{source_match}", "tests",
                    {"strategy": "path_mirror"},
                ))

    # Strategy 2: Import-based (test imports source module)
    file_import_targets = defaultdict(set)
    for edge in edges:
        if edge.type == "imports" and edge.source.startswith("file:") and edge.target.startswith("file:"):
            src = edge.source[5:]
            tgt = edge.target[5:]
            if is_test_file(src) and tgt in source_files:
                file_import_targets[src].add(tgt)

    for test_file, source_targets in file_import_targets.items():
        for source_target in source_targets:
            key = (f"file:{test_file}", f"file:{source_target}")
            if key not in seen:
                seen.add(key)
                test_edges.append(Edge(
                    f"file:{test_file}", f"file:{source_target}", "tests",
                    {"strategy": "import"},
                ))

    # Strategy 3: Name matching (test_foo -> foo, TestUserModel -> UserModel)
    for sym in symbols:
        if not sym.is_test:
            continue
        if sym.type not in ("function", "method", "class"):
            continue

        targets = _match_test_symbol_to_source(sym, symbol_by_name)
        for target_sym in targets:
            key = (sym.id, target_sym.id)
            if key not in seen:
                seen.add(key)
                test_edges.append(Edge(
                    sym.id, target_sym.id, "tests",
                    {"strategy": "name_match"},
                ))

    return test_edges


def _match_test_to_source_path(test_path: str, source_files: set) -> Optional[str]:
    """Match a test file to its source file by path conventions."""
    test_p = Path(test_path)
    stem = test_p.stem
    parent = str(test_p.parent)
    suffix = test_p.suffix

    # Strip test prefixes/suffixes from filename
    source_stem = stem
    for prefix in ("test_", "test"):
        if source_stem.startswith(prefix):
            source_stem = source_stem[len(prefix):]
            break
    for suffix_str in ("_test", "_spec", ".test", ".spec"):
        if source_stem.endswith(suffix_str):
            source_stem = source_stem[:-len(suffix_str)]
            break

    if source_stem == stem:
        return None  # No transformation happened

    # Try same directory
    candidate = str(test_p.with_name(source_stem + suffix))
    if candidate in source_files:
        return candidate

    # Try stripping test directory prefixes
    parent_parts = Path(parent).parts
    for test_dir in ("tests", "test", "__tests__", "spec"):
        if test_dir in parent_parts:
            idx = parent_parts.index(test_dir)
            # Replace test dir with src/lib or just remove it
            for src_dir in ("src", "lib", "pkg", "internal", ""):
                if src_dir:
                    new_parts = parent_parts[:idx] + (src_dir,) + parent_parts[idx + 1:]
                else:
                    new_parts = parent_parts[:idx] + parent_parts[idx + 1:]
                candidate = str(Path(*new_parts) / (source_stem + suffix)) if new_parts else source_stem + suffix
                if candidate in source_files:
                    return candidate

    # Try searching source_files for matching stem
    for sf in source_files:
        if Path(sf).stem == source_stem:
            return sf

    return None


def _match_test_symbol_to_source(
    test_sym: Symbol, symbol_by_name: dict,
) -> list[Symbol]:
    """Match a test function/class to the source symbol it tests."""
    name = test_sym.name
    matches = []

    # test_foo -> foo
    if name.startswith("test_"):
        target_name = name[5:]
        if target_name in symbol_by_name:
            matches.extend(symbol_by_name[target_name])

    # TestFoo -> Foo  (test classes)
    if name.startswith("Test") and len(name) > 4 and name[4].isupper():
        target_name = name[4:]
        if target_name in symbol_by_name:
            matches.extend(symbol_by_name[target_name])

    # test_ClassName_method -> ClassName.method
    if name.startswith("test_") and "_" in name[5:]:
        parts = name[5:].split("_", 1)
        if len(parts) == 2:
            class_name, method_hint = parts
            qualified = f"{class_name}.{method_hint}"
            if qualified in symbol_by_name:
                matches.extend(symbol_by_name[qualified])

    return matches


# ── Graph Builder ─────────────────────────────────────────────────

def resolve_imports(
    symbols: list[Symbol], edges: list[Edge], file_map: dict[str, str],
    module_boundaries: list[ModuleBoundary],
) -> list[Edge]:
    """Resolve import edges to actual files where possible."""
    resolved = []

    # Build lookup maps
    module_to_files = defaultdict(list)
    for rel_path, lang in file_map.items():
        parts = Path(rel_path).with_suffix("").parts
        module_name = ".".join(parts)
        module_to_files[module_name].append(rel_path)

        for i in range(len(parts)):
            partial = ".".join(parts[i:])
            module_to_files[partial].append(rel_path)

    # Also index by __init__.py aware paths
    for rel_path, lang in file_map.items():
        if Path(rel_path).stem == "__init__":
            # Package directory maps to its __init__.py
            pkg_parts = Path(rel_path).parent.parts
            if pkg_parts:
                pkg_name = ".".join(pkg_parts)
                module_to_files[pkg_name].append(rel_path)

    # Map Go import paths
    go_pkg_map = defaultdict(list)
    for rel_path, lang in file_map.items():
        if lang == "go":
            pkg_dir = str(Path(rel_path).parent)
            go_pkg_map[pkg_dir].append(rel_path)

    # Use module boundaries for better Go module resolution
    go_module_prefix = ""
    for boundary in module_boundaries:
        if boundary.language == "go" and boundary.name:
            go_module_prefix = boundary.name
            break

    # Build (file, name) -> symbol index for O(1) symbol lookup in import resolution
    file_name_to_symbol = {}
    for sym in symbols:
        file_name_to_symbol[(sym.file, sym.name)] = sym

    for edge in edges:
        if edge.type != "imports":
            resolved.append(edge)
            continue

        target = edge.target
        if target.startswith("module:"):
            module_name = target[7:]

            # Resolve relative imports (./foo, ../bar, or Python relative like pkg.sub)
            # against the importing file's directory
            if module_name.startswith("."):
                src_file = edge.source[5:] if edge.source.startswith("file:") else ""
                if src_file:
                    src_lang = file_map.get(src_file, "")
                    src_dir = str(Path(src_file).parent)
                    rel_target = os.path.normpath(os.path.join(src_dir, module_name))
                    found = False
                    # Build candidate extensions based on language
                    if src_lang in ("javascript", "typescript"):
                        exts = ("", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".mts")
                        index_names = ("index.ts", "index.tsx", "index.js", "index.jsx")
                    elif src_lang == "python":
                        exts = ("", ".py")
                        index_names = ("__init__.py",)
                    else:
                        exts = ("",)
                        index_names = ()
                    for ext in exts:
                        candidate = rel_target + ext
                        if candidate in file_map:
                            resolved.append(Edge(edge.source, f"file:{candidate}", "imports", edge.metadata))
                            found = True
                            break
                    if not found:
                        for index_name in index_names:
                            candidate = os.path.join(rel_target, index_name)
                            if candidate in file_map:
                                resolved.append(Edge(edge.source, f"file:{candidate}", "imports", edge.metadata))
                                found = True
                                break
                    if not found:
                        resolved.append(edge)  # Keep unresolved
                    continue

            # Try direct file mapping
            candidates = module_to_files.get(module_name, [])
            if candidates:
                for c in candidates:
                    resolved.append(Edge(edge.source, f"file:{c}", "imports", edge.metadata))
                continue

            # Try Go package mapping with module prefix stripping
            if go_module_prefix and module_name.startswith(go_module_prefix):
                local_path = module_name[len(go_module_prefix):].lstrip("/")
                if local_path in go_pkg_map:
                    for f in go_pkg_map[local_path]:
                        resolved.append(Edge(edge.source, f"file:{f}", "imports", edge.metadata))
                    continue

            # Try Go package mapping by suffix
            for pkg_path, pkg_files in go_pkg_map.items():
                if module_name.endswith(pkg_path.replace("/", ".")):
                    for f in pkg_files:
                        resolved.append(Edge(edge.source, f"file:{f}", "imports", edge.metadata))
                    break
            else:
                resolved.append(edge)  # Keep unresolved external

        elif target.startswith("symbol:"):
            symbol_path = target[7:]
            parts = symbol_path.rsplit(".", 1)
            if len(parts) == 2:
                module_part, sym_name = parts
                candidates = module_to_files.get(module_part, [])
                if candidates:
                    for c in candidates:
                        sym = file_name_to_symbol.get((c, sym_name))
                        if sym:
                            resolved.append(Edge(edge.source, sym.id, "imports", edge.metadata))
                        else:
                            resolved.append(Edge(edge.source, f"file:{c}", "imports", edge.metadata))
                    continue
            resolved.append(edge)
        else:
            resolved.append(edge)

    # Build class/struct/interface name -> symbols for O(1) inheritance resolution
    # Use defaultdict(list) to handle multiple classes with the same name across files
    class_by_name: dict[str, list[Symbol]] = defaultdict(list)
    for sym in symbols:
        if sym.type in ("class", "struct", "interface"):
            class_by_name[sym.name].append(sym)

    # Resolve inheritance edges
    final = []
    for edge in resolved:
        if edge.type == "inherits" and edge.metadata.get("unresolved"):
            base_name = edge.target.split(":")[-1]
            candidates = class_by_name.get(base_name, [])
            if len(candidates) == 1:
                final.append(Edge(edge.source, candidates[0].id, "inherits"))
            elif candidates:
                # Disambiguate: prefer class in the same file as the source symbol
                source_file = edge.source.split(":")[1] if ":" in edge.source else ""
                match = next((c for c in candidates if c.file == source_file), candidates[0])
                final.append(Edge(edge.source, match.id, "inherits"))
            else:
                final.append(edge)
        else:
            final.append(edge)

    return final


def compute_directory_stats(
    files: list[tuple[str, str, str, bool]], symbols: list[Symbol],
) -> dict:
    """Compute statistics per directory."""
    dir_stats = defaultdict(lambda: {
        "files": 0, "test_files": 0, "languages": set(),
        "symbols": 0, "classes": 0, "functions": 0,
    })

    for _, rel_path, lang, is_test in files:
        d = str(Path(rel_path).parent) or "."
        dir_stats[d]["files"] += 1
        if is_test:
            dir_stats[d]["test_files"] += 1
        dir_stats[d]["languages"].add(lang)

    for sym in symbols:
        d = str(Path(sym.file).parent) or "."
        dir_stats[d]["symbols"] += 1
        if sym.type in ("class", "struct", "interface"):
            dir_stats[d]["classes"] += 1
        elif sym.type in ("function", "method"):
            dir_stats[d]["functions"] += 1

    for d in dir_stats:
        dir_stats[d]["languages"] = sorted(dir_stats[d]["languages"])

    return dict(dir_stats)


# ── Output Generators ─────────────────────────────────────────────

def _serialize_boundaries(boundaries: list[ModuleBoundary]) -> list[dict]:
    """Serialize module boundaries to dicts (shared by graph and modules output)."""
    result = []
    for mb in boundaries:
        entry = {"root_dir": mb.root_dir, "build_file": mb.build_file}
        if mb.name:
            entry["name"] = mb.name
        if mb.language:
            entry["language"] = mb.language
        result.append(entry)
    return result

def compute_graph_stats(
    symbols: list[Symbol], edges: list[Edge],
    files: list[tuple[str, str, str, bool]],
) -> dict:
    """Compute stats shared by graph.json and summary.md (single pass)."""
    lang_counts = Counter(lang for _, _, lang, _ in files)
    type_counts = Counter(sym.type for sym in symbols)
    edge_type_counts = Counter(edge.type for edge in edges)
    test_count = sum(1 for _, _, _, t in files if t)
    return {
        "lang_counts": lang_counts,
        "type_counts": type_counts,
        "edge_type_counts": edge_type_counts,
        "test_count": test_count,
    }


def generate_graph_json(
    symbols: list[Symbol], edges: list[Edge],
    files: list[tuple[str, str, str, bool]], repo_root: str,
    file_hashes: dict, module_boundaries: list[ModuleBoundary],
    line_counts: Optional[dict] = None,
    stats: Optional[dict] = None,
) -> dict:
    """Generate the full knowledge graph as a JSON-serializable dict."""
    if stats is None:
        stats = compute_graph_stats(symbols, edges, files)

    lang_counts = stats["lang_counts"]
    type_counts = stats["type_counts"]
    edge_type_counts = stats["edge_type_counts"]

    if line_counts is None:
        line_counts = {}

    # Build file nodes
    nodes = []
    for abs_path, rel_path, lang, is_test in files:
        line_count = line_counts.get(rel_path, 0)

        node = {
            "id": f"file:{rel_path}",
            "type": "file",
            "name": Path(rel_path).name,
            "path": rel_path,
            "language": lang,
            "line_count": line_count,
        }
        if is_test:
            node["is_test"] = True
        nodes.append(node)

    # Add symbol nodes
    for sym in symbols:
        node = asdict(sym)
        node = {k: v for k, v in node.items() if v is not None and v != [] and v is not False}
        if "exported" not in node:
            node["exported"] = False
        if sym.is_test:
            node["is_test"] = True
        nodes.append(node)

    # Serialize edges
    edge_list = [asdict(e) for e in edges]
    for e in edge_list:
        if not e.get("metadata"):
            del e["metadata"]

    return {
        "metadata": {
            "repo_root": repo_root,
            "generated_at": datetime.now(timezone.utc).isoformat(),
            "languages": dict(sorted(lang_counts.items(), key=lambda x: -x[1])),
            "total_files": len(files),
            "total_symbols": len(symbols),
            "symbol_types": dict(sorted(type_counts.items(), key=lambda x: -x[1])),
            "edge_types": dict(sorted(edge_type_counts.items(), key=lambda x: -x[1])),
            "file_hashes": file_hashes,
            "module_boundaries": _serialize_boundaries(module_boundaries),
        },
        "nodes": nodes,
        "edges": edge_list,
    }


def generate_tags_json(symbols: list[Symbol]) -> list[dict]:
    """Generate a flat symbol index for quick lookup."""
    tags = []
    for sym in symbols:
        tag = {
            "name": sym.name,
            "type": sym.type,
            "file": sym.file,
            "line": sym.line_start,
            "end_line": sym.line_end,
            "language": sym.language,
        }
        if sym.scope:
            tag["scope"] = sym.scope
        if sym.params:
            tag["params"] = sym.params
        if sym.bases:
            tag["bases"] = sym.bases
        if sym.docstring:
            tag["doc"] = sym.docstring
        if sym.signature:
            tag["signature"] = sym.signature
        if not sym.exported:
            tag["private"] = True
        if sym.is_test:
            tag["is_test"] = True
        tags.append(tag)
    return sorted(tags, key=lambda t: (t["file"], t["line"]))


def generate_modules_json(
    edges: list[Edge], files: list[tuple[str, str, str, bool]],
    module_boundaries: list[ModuleBoundary],
) -> dict:
    """Generate module-level dependency map with build module awareness."""
    # Group files by directory
    dir_files = defaultdict(list)
    for _, rel_path, lang, _ in files:
        d = str(Path(rel_path).parent) or "."
        dir_files[d].append(rel_path)

    file_to_dir = {}
    for _, rel_path, _, _ in files:
        file_to_dir[rel_path] = str(Path(rel_path).parent) or "."

    # Module dependency edges
    module_deps = defaultdict(set)
    for edge in edges:
        if edge.type != "imports":
            continue
        src_file = edge.source.replace("file:", "") if edge.source.startswith("file:") else None
        tgt_file = edge.target.replace("file:", "") if edge.target.startswith("file:") else None

        if src_file and tgt_file and src_file in file_to_dir and tgt_file in file_to_dir:
            src_dir = file_to_dir[src_file]
            tgt_dir = file_to_dir[tgt_file]
            if src_dir != tgt_dir:
                module_deps[src_dir].add(tgt_dir)

    # Test relationships per module
    module_tests = defaultdict(set)
    for edge in edges:
        if edge.type == "tests":
            src = edge.source.replace("file:", "") if edge.source.startswith("file:") else None
            tgt = edge.target.replace("file:", "") if edge.target.startswith("file:") else None
            if src and tgt and src in file_to_dir and tgt in file_to_dir:
                test_dir = file_to_dir[src]
                source_dir = file_to_dir[tgt]
                module_tests[source_dir].add(test_dir)

    # Build module map
    module_map = {}
    for d, f in sorted(dir_files.items()):
        entry = {
            "files": sorted(f),
            "depends_on": sorted(module_deps.get(d, set())),
        }
        tests = module_tests.get(d, set())
        if tests:
            entry["tested_by"] = sorted(tests)
        module_map[d] = entry

    return {
        "modules": module_map,
        "build_modules": _serialize_boundaries(module_boundaries),
    }


def generate_summary_md(
    symbols: list[Symbol], edges: list[Edge],
    files: list[tuple[str, str, str, bool]], dir_stats: dict,
    repo_root: str, ts_available: bool,
    stats: Optional[dict] = None,
    module_deps: Optional[dict] = None,
) -> str:
    """Generate a human/AI-readable codebase summary."""
    if stats is None:
        stats = compute_graph_stats(symbols, edges, files)

    lang_counts = stats["lang_counts"]
    type_counts = stats["type_counts"]
    edge_type_counts = stats["edge_type_counts"]
    test_count = stats["test_count"]

    # Single pass over edges for hub files, hub symbols, inheritance, and test coverage
    import_counts = Counter()
    call_counts = Counter()
    inheritance = defaultdict(list)
    tested_files = set()
    test_map = defaultdict(set)

    for edge in edges:
        if edge.type == "imports" and edge.target.startswith("file:"):
            import_counts[edge.target[5:]] += 1
        elif edge.type == "calls":
            call_counts[edge.target] += 1
        elif edge.type == "inherits" and not edge.metadata.get("unresolved"):
            child_name = edge.source.split(":")[-1]
            parent_name = edge.target.split(":")[-1]
            inheritance[parent_name].append(child_name)
        elif edge.type == "tests" and edge.target.startswith("file:"):
            tested_files.add(edge.target[5:])
            test_name = edge.source[5:] if edge.source.startswith("file:") else edge.source
            test_map[edge.target[5:]].add(test_name)

    hub_files = import_counts.most_common(15)
    hub_symbols = call_counts.most_common(15)

    lines = []
    lines.append("# Codebase Knowledge Graph\n")
    lines.append(f"Generated: {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M UTC')}")
    lines.append(f"Parser: {'Tree-sitter + AST' if ts_available else 'AST + Regex'}\n")

    # Statistics
    lines.append("## Statistics\n")
    lines.append(f"- **{len(files)}** files ({test_count} test files) across **{len(lang_counts)}** languages")
    lines.append(f"- **{len(symbols)}** symbols extracted")
    for st, count in sorted(type_counts.items(), key=lambda x: -x[1]):
        plural = st if st.endswith("s") else f"{st}es" if st.endswith("ss") else f"{st}s"
        lines.append(f"  - {count} {plural if count != 1 else st}")
    lines.append(f"- **{len(edges)}** edges:")
    for et, count in sorted(edge_type_counts.items(), key=lambda x: -x[1]):
        lines.append(f"  - {count} {et}")
    if tested_files:
        source_count = sum(1 for _, _, _, t in files if not t)
        lines.append(f"- **{len(tested_files)}/{source_count}** source files have test coverage")
    lines.append("")

    # Languages
    lines.append("## Languages\n")
    lines.append("| Language | Files |")
    lines.append("|----------|-------|")
    for lang, count in sorted(lang_counts.items(), key=lambda x: -x[1]):
        lines.append(f"| {lang} | {count} |")
    lines.append("")

    # Directory structure
    lines.append("## Directory Structure\n")
    lines.append("```")
    for d in sorted(dir_stats.keys()):
        dir_info = dir_stats[d]
        indent = "  " * d.count("/")
        dir_name = Path(d).name or "."
        lang_str = ", ".join(dir_info["languages"])
        test_str = f", {dir_info['test_files']} tests" if dir_info.get("test_files") else ""
        lines.append(f"{indent}{dir_name}/ ({dir_info['files']} files{test_str}, {lang_str})")
    lines.append("```\n")

    # Hub files
    if hub_files:
        lines.append("## Hub Files (most imported)\n")
        lines.append("| File | Imported by |")
        lines.append("|------|-------------|")
        for f, count in hub_files:
            lines.append(f"| `{f}` | {count} files |")
        lines.append("")

    # Most-called symbols
    if hub_symbols:
        lines.append("## Most-Called Symbols\n")
        lines.append("| Symbol | Called by |")
        lines.append("|--------|----------|")
        for sym_id, count in hub_symbols:
            parts = sym_id.split(":", 2)
            display = parts[-1] if len(parts) >= 3 else sym_id
            lines.append(f"| `{display}` | {count} callers |")
        lines.append("")

    # Key classes
    classes = [s for s in symbols if s.type in ("class", "struct", "interface") and s.exported and not s.is_test]
    if classes:
        lines.append("## Key Types\n")
        lines.append("| Type | Kind | File | Line | Bases |")
        lines.append("|------|------|------|------|-------|")
        for cls in sorted(classes, key=lambda c: c.name)[:50]:
            bases_str = ", ".join(cls.bases) if cls.bases else "-"
            lines.append(f"| `{cls.name}` | {cls.type} | `{cls.file}` | {cls.line_start} | {bases_str} |")
        if len(classes) > 50:
            lines.append(f"\n*... and {len(classes) - 50} more types (see tags.json)*\n")
        lines.append("")

    # Inheritance trees
    roots = [name for name in inheritance if not any(name in children for children in inheritance.values())]
    if roots:
        lines.append("## Inheritance Hierarchy\n")
        lines.append("```")

        def print_tree(name, indent=0, visited=None):
            if visited is None:
                visited = set()
            if name in visited:
                return
            visited.add(name)
            prefix = "  " * indent + ("|- " if indent > 0 else "")
            lines.append(f"{prefix}{name}")
            for child in sorted(inheritance.get(name, [])):
                print_tree(child, indent + 1, visited)

        for root in sorted(roots)[:10]:
            print_tree(root)
        lines.append("```\n")

    # Key functions
    exported_funcs = [s for s in symbols if s.type == "function" and s.exported and not s.is_test]
    if exported_funcs:
        lines.append("## Key Functions\n")
        lines.append("| Function | File | Line | Params |")
        lines.append("|----------|------|------|--------|")
        for func in sorted(exported_funcs, key=lambda f: f.name)[:50]:
            params_str = ", ".join(func.params[:5]) if func.params else "-"
            if len(func.params) > 5:
                params_str += ", ..."
            lines.append(f"| `{func.name}` | `{func.file}` | {func.line_start} | {params_str} |")
        if len(exported_funcs) > 50:
            lines.append(f"\n*... and {len(exported_funcs) - 50} more functions (see tags.json)*\n")
        lines.append("")

    # Module dependencies (use pre-computed or compute from modules_json)
    if module_deps is not None:
        dir_deps = module_deps
    else:
        file_set = {rel for _, rel, _, _ in files}
        dir_deps = defaultdict(set)
        for edge in edges:
            if edge.type == "imports":
                src = edge.source.replace("file:", "")
                tgt = edge.target.replace("file:", "")
                if src in file_set and tgt in file_set:
                    sd = str(Path(src).parent) or "."
                    td = str(Path(tgt).parent) or "."
                    if sd != td:
                        dir_deps[sd].add(td)

    if dir_deps:
        lines.append("## Module Dependencies\n")
        lines.append("```")
        for src_dir in sorted(dir_deps.keys()):
            deps = sorted(dir_deps[src_dir])
            lines.append(f"{src_dir} -> {', '.join(deps)}")
        lines.append("```\n")

    # Test coverage (test_map already built in single-pass above)
    if tested_files:
        lines.append("## Test Coverage Map\n")
        lines.append("| Source File | Tested By |")
        lines.append("|------------|-----------|")
        for src_file in sorted(test_map.keys())[:30]:
            testers = sorted(test_map[src_file])
            testers_str = ", ".join(f"`{t}`" for t in testers[:3])
            if len(testers) > 3:
                testers_str += f" +{len(testers) - 3} more"
            lines.append(f"| `{src_file}` | {testers_str} |")
        if len(test_map) > 30:
            lines.append(f"\n*... and {len(test_map) - 30} more tested files*\n")
        lines.append("")

    return "\n".join(lines)


def _edge_file(node_id: str) -> str:
    """Extract the file path from a node ID for incremental merge filtering."""
    if node_id.startswith("file:"):
        return node_id[5:]
    # Symbol IDs have format "type:file:name" — extract file part
    parts = node_id.split(":", 2)
    if len(parts) >= 3:
        return parts[1]
    return ""


# ── Main ──────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Build a knowledge graph of a codebase with call graph and test mapping",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "--repo-root", default=".",
        help="Repository root directory (default: current directory)",
    )
    parser.add_argument(
        "--output-dir", default=None,
        help="Output directory (default: <repo-root>/docs/code-tree)",
    )
    parser.add_argument(
        "--exclude", action="append", default=[],
        help="Additional glob patterns to exclude (repeatable)",
    )
    parser.add_argument(
        "--max-file-size", type=int, default=MAX_FILE_SIZE,
        help=f"Skip files larger than this in bytes (default: {MAX_FILE_SIZE})",
    )
    parser.add_argument(
        "--exclude-tests", action="store_true",
        help="Exclude test files from analysis (tests are included by default)",
    )
    parser.add_argument(
        "--no-tree-sitter", action="store_true",
        help="Disable tree-sitter even if available",
    )
    parser.add_argument(
        "--no-call-graph", action="store_true",
        help="Skip call graph extraction (faster but less detailed)",
    )
    parser.add_argument(
        "--incremental", action="store_true",
        help="Only reparse files that changed since last run",
    )
    parser.add_argument(
        "--quiet", "-q", action="store_true",
        help="Suppress progress output",
    )

    args = parser.parse_args()
    repo_root = os.path.abspath(args.repo_root)
    output_dir = args.output_dir or os.path.join(repo_root, "docs", "code-tree")
    excludes = DEFAULT_EXCLUDES + args.exclude
    include_tests = not args.exclude_tests

    def log(msg):
        if not args.quiet:
            print(msg, file=sys.stderr)

    log(f"Scanning: {repo_root}")

    # Load previous graph for incremental mode
    prev_hashes = {}
    prev_graph = None
    if args.incremental:
        prev_graph_path = os.path.join(output_dir, "graph.json")
        if os.path.exists(prev_graph_path):
            try:
                with open(prev_graph_path, "r", encoding="utf-8") as f:
                    prev_graph = json.load(f)
                prev_hashes = prev_graph.get("metadata", {}).get("file_hashes", {})
                log(f"Loaded previous graph with {len(prev_hashes)} file hashes")
            except (json.JSONDecodeError, OSError):
                prev_graph = None

    # Load tree-sitter
    ts_parsers = {}
    if not args.no_tree_sitter:
        ts_parsers = try_load_tree_sitter()
        if ts_parsers:
            log(f"Tree-sitter available for: {', '.join(sorted(ts_parsers.keys()))}")
        else:
            log("Tree-sitter not available, using AST/regex parsers")

    # Discover files and build boundaries in a single pass
    files, module_boundaries = discover_all(repo_root, excludes, include_tests, args.max_file_size)
    test_file_count = sum(1 for _, _, _, t in files if t)
    source_file_count = len(files) - test_file_count
    log(f"Found {len(files)} files ({source_file_count} source, {test_file_count} test)")

    if not files:
        log("No files found. Check --repo-root and --exclude options.")
        sys.exit(1)

    if module_boundaries:
        log(f"Found {len(module_boundaries)} build module(s): {', '.join(mb.name or mb.build_file for mb in module_boundaries)}")

    # Compute file hashes and determine what needs parsing
    current_hashes = {}
    files_to_parse = []
    for abs_path, rel_path, lang, is_test in files:
        h = file_hash(abs_path)
        current_hashes[rel_path] = h
        if args.incremental and prev_hashes.get(rel_path) == h:
            continue  # File unchanged
        files_to_parse.append((abs_path, rel_path, lang, is_test))

    if args.incremental:
        log(f"Incremental: {len(files_to_parse)}/{len(files)} files need parsing")

    # Build file_map from ALL files (needed by resolve_imports even in incremental mode)
    file_map = {rel_path: lang for _, rel_path, lang, _ in files}

    # Parse files
    all_symbols = []
    all_edges = []
    all_call_refs = []
    line_counts = {}

    # In incremental mode, load symbols/edges for unchanged files from previous graph
    changed_files = {rel for _, rel, _, _ in files_to_parse} if files_to_parse else set()
    if args.incremental and prev_graph is not None and changed_files:
        prev_nodes = prev_graph.get("nodes", [])
        prev_edges = prev_graph.get("edges", [])

        for node in prev_nodes:
            node_file = node.get("file") or node.get("path", "")
            if node_file and node_file not in changed_files:
                if node.get("type") == "file":
                    line_counts[node.get("path", "")] = node.get("line_count", 0)
                else:
                    all_symbols.append(Symbol(
                        id=node["id"], type=node["type"], name=node["name"],
                        file=node.get("file", ""), language=node.get("language", ""),
                        line_start=node.get("line_start", 0),
                        line_end=node.get("line_end", 0),
                        scope=node.get("scope"),
                        params=node.get("params", []),
                        bases=node.get("bases", []),
                        decorators=node.get("decorators", []),
                        docstring=node.get("docstring"),
                        exported=node.get("exported", True),
                        signature=node.get("signature"),
                        is_test=node.get("is_test", False),
                    ))

        for edge_data in prev_edges:
            src_file = _edge_file(edge_data.get("source", ""))
            # Keep edges whose source is from an unchanged file (changed files
            # will be reparsed and their outgoing edges re-emitted fresh)
            if src_file not in changed_files:
                all_edges.append(Edge(
                    source=edge_data["source"], target=edge_data["target"],
                    type=edge_data["type"],
                    metadata=edge_data.get("metadata", {}),
                ))

        log(f"Loaded {len(all_symbols)} symbols, {len(all_edges)} edges from previous graph (unchanged files)")

    parse_target = files_to_parse if files_to_parse else files
    parse_count = len(parse_target)

    for i, (abs_path, rel_path, lang, is_test) in enumerate(parse_target):
        if not args.quiet and (i + 1) % 100 == 0:
            log(f"  Parsed {i + 1}/{parse_count} files...")

        symbols, file_edges, call_refs, lc = parse_file(abs_path, rel_path, lang, ts_parsers, is_test)
        line_counts[rel_path] = lc
        all_symbols.extend(symbols)
        all_edges.extend(file_edges)
        if not args.no_call_graph:
            all_call_refs.extend(call_refs)

    log(f"Extracted {len(all_symbols)} symbols, {len(all_edges)} edges, {len(all_call_refs)} call references")

    # Resolve calls before import normalization — resolve_calls() needs the
    # raw module:/symbol: import edges to rebuild alias mappings.  Once
    # resolve_imports() rewrites those targets to file: paths the alias
    # information is lost.
    if not args.no_call_graph and all_call_refs:
        call_edges = resolve_calls(all_call_refs, all_symbols, all_edges, file_map)
        all_edges.extend(call_edges)
        log(f"Resolved {len(call_edges)} call edges from {len(all_call_refs)} references")

    # Resolve imports (rewrites module:/symbol: targets to file: paths)
    all_edges = resolve_imports(all_symbols, all_edges, file_map, module_boundaries)
    log(f"Resolved imports: {len(all_edges)} edges after resolution")

    # Detect test-to-code relationships
    if include_tests:
        test_edges = detect_test_relationships(all_symbols, all_edges, files, file_map)
        all_edges.extend(test_edges)
        log(f"Mapped {len(test_edges)} test relationships")

    # Compute directory stats
    dir_stats = compute_directory_stats(files, all_symbols)

    # Generate outputs
    os.makedirs(output_dir, exist_ok=True)

    # Compute stats once and share across generators
    graph_stats = compute_graph_stats(all_symbols, all_edges, files)

    # graph.json
    graph = generate_graph_json(
        all_symbols, all_edges, files, repo_root, current_hashes, module_boundaries,
        line_counts=line_counts, stats=graph_stats,
    )
    graph_path = os.path.join(output_dir, "graph.json")
    with open(graph_path, "w", encoding="utf-8") as f:
        json.dump(graph, f, indent=2, default=str)
    log(f"Wrote {graph_path}")

    # tags.json
    tags = generate_tags_json(all_symbols)
    tags_path = os.path.join(output_dir, "tags.json")
    with open(tags_path, "w", encoding="utf-8") as f:
        json.dump(tags, f, indent=2)
    log(f"Wrote {tags_path}")

    # modules.json
    modules = generate_modules_json(all_edges, files, module_boundaries)
    modules_path = os.path.join(output_dir, "modules.json")
    with open(modules_path, "w", encoding="utf-8") as f:
        json.dump(modules, f, indent=2)
    log(f"Wrote {modules_path}")

    # summary.md (reuse stats)
    summary = generate_summary_md(
        all_symbols, all_edges, files, dir_stats,
        repo_root, bool(ts_parsers), stats=graph_stats,
    )
    summary_path = os.path.join(output_dir, "summary.md")
    with open(summary_path, "w", encoding="utf-8") as f:
        f.write(summary)
    log(f"Wrote {summary_path}")

    # Print summary (reuse stats instead of re-counting)
    log(f"\nDone. Output in: {output_dir}")
    log(f"  graph.json   - Full knowledge graph ({len(graph['nodes'])} nodes, {len(graph['edges'])} edges)")
    log(f"  tags.json    - Symbol index ({len(tags)} tags)")
    log(f"  modules.json - Module dependencies ({len(modules['modules'])} modules)")
    log(f"  summary.md   - Codebase overview")
    log(f"\n  Edge breakdown:")
    for et, count in sorted(graph_stats["edge_type_counts"].items(), key=lambda x: -x[1]):
        log(f"    {et}: {count}")


if __name__ == "__main__":
    main()
