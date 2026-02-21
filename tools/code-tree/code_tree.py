#!/usr/bin/env python3
"""code_tree.py - Build a knowledge graph of a codebase.

Hybrid Parser + Knowledge Graph + Chunked Context approach:
1. Parse: Extract symbols (classes, functions, imports) from every file
2. Index: Link symbols into a knowledge graph with dependency edges
3. Output: Generate structured artifacts for AI agent consumption

Usage:
    python code_tree.py [--repo-root .] [--output-dir docs/code-tree]
    python code_tree.py --help

Zero external dependencies required. Optional: tree-sitter for enhanced parsing.
"""

import argparse
import ast as python_ast
import fnmatch
import json
import os
import re
import sys
from collections import defaultdict
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

MAX_FILE_SIZE = 1_048_576  # 1 MB


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
    params: list = field(default_factory=list)
    bases: list = field(default_factory=list)
    decorators: list = field(default_factory=list)
    docstring: Optional[str] = None
    exported: bool = True
    signature: Optional[str] = None


@dataclass
class Edge:
    source: str
    target: str
    type: str  # contains, imports, inherits, implements
    metadata: dict = field(default_factory=dict)


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


def discover_files(
    root: str,
    excludes: list[str],
    include_tests: bool = False,
    max_file_size: int = MAX_FILE_SIZE,
) -> list[tuple[str, str, str]]:
    """Discover source files. Returns list of (abs_path, rel_path, language)."""
    test_markers = {"test_", "_test.", ".test.", ".spec.", "tests/", "__tests__/", "test/"}
    results = []
    root = os.path.abspath(root)

    for dirpath, dirnames, filenames in os.walk(root):
        # Prune excluded directories in-place
        dirnames[:] = [
            d for d in dirnames
            if d not in excludes and not d.startswith(".")
        ]

        for fname in filenames:
            abs_path = os.path.join(dirpath, fname)
            rel_path = os.path.relpath(abs_path, root)

            if should_exclude(rel_path, excludes):
                continue

            ext = Path(fname).suffix.lower()
            lang = LANG_EXTENSIONS.get(ext)
            if not lang:
                continue

            if not include_tests:
                lower_rel = rel_path.lower()
                if any(m in lower_rel for m in test_markers):
                    continue

            try:
                size = os.path.getsize(abs_path)
                if size > max_file_size or size == 0:
                    continue
            except OSError:
                continue

            if is_binary(abs_path):
                continue

            results.append((abs_path, rel_path, lang))

    return sorted(results, key=lambda x: x[1])


def read_file(path: str) -> Optional[str]:
    for encoding in ("utf-8", "latin-1"):
        try:
            with open(path, "r", encoding=encoding) as f:
                return f.read()
        except (UnicodeDecodeError, OSError):
            continue
    return None


# ── Python Parser (ast module) ────────────────────────────────────

def parse_python(content: str, rel_path: str) -> tuple[list[Symbol], list[Edge]]:
    symbols = []
    edges = []
    file_id = f"file:{rel_path}"

    try:
        tree = python_ast.parse(content, filename=rel_path)
    except SyntaxError:
        return symbols, edges

    for node in python_ast.iter_child_nodes(tree):
        if isinstance(node, python_ast.ClassDef):
            _extract_python_class(node, rel_path, file_id, symbols, edges)
        elif isinstance(node, (python_ast.FunctionDef, python_ast.AsyncFunctionDef)):
            _extract_python_function(node, rel_path, file_id, symbols, edges)
        elif isinstance(node, python_ast.Import):
            for alias in node.names:
                edges.append(Edge(file_id, f"module:{alias.name}", "imports"))
        elif isinstance(node, python_ast.ImportFrom):
            if node.module:
                edges.append(Edge(file_id, f"module:{node.module}", "imports"))
                for alias in (node.names or []):
                    if alias.name != "*":
                        edges.append(Edge(
                            file_id,
                            f"symbol:{node.module}.{alias.name}",
                            "imports",
                            {"from_import": True},
                        ))
        elif isinstance(node, python_ast.Assign):
            # Module-level constants (UPPER_CASE)
            for target in node.targets:
                if isinstance(target, python_ast.Name) and target.id.isupper():
                    sym_id = f"constant:{rel_path}:{target.id}"
                    symbols.append(Symbol(
                        id=sym_id, type="constant", name=target.id,
                        file=rel_path, language="python",
                        line_start=node.lineno,
                        line_end=node.end_lineno or node.lineno,
                        exported=not target.id.startswith("_"),
                    ))
                    edges.append(Edge(file_id, sym_id, "contains"))

    return symbols, edges


def _extract_python_class(
    node: python_ast.ClassDef, rel_path: str, file_id: str,
    symbols: list, edges: list,
):
    sym_id = f"class:{rel_path}:{node.name}"
    bases = []
    for base in node.bases:
        try:
            bases.append(python_ast.unparse(base))
        except Exception:
            if isinstance(base, python_ast.Name):
                bases.append(base.id)

    decorators = []
    for dec in node.decorator_list:
        try:
            decorators.append(python_ast.unparse(dec))
        except Exception:
            pass

    docstring = python_ast.get_docstring(node)
    symbols.append(Symbol(
        id=sym_id, type="class", name=node.name,
        file=rel_path, language="python",
        line_start=node.lineno,
        line_end=node.end_lineno or node.lineno,
        bases=bases, decorators=decorators,
        docstring=docstring[:200] if docstring else None,
        exported=not node.name.startswith("_"),
    ))
    edges.append(Edge(file_id, sym_id, "contains"))

    for base_name in bases:
        edges.append(Edge(sym_id, f"class:?:{base_name}", "inherits", {"unresolved": True}))

    # Extract methods
    for item in python_ast.iter_child_nodes(node):
        if isinstance(item, (python_ast.FunctionDef, python_ast.AsyncFunctionDef)):
            method_id = f"method:{rel_path}:{node.name}.{item.name}"
            params = [a.arg for a in item.args.args if a.arg != "self"]
            mdoc = python_ast.get_docstring(item)
            m_decorators = []
            for dec in item.decorator_list:
                try:
                    m_decorators.append(python_ast.unparse(dec))
                except Exception:
                    pass

            symbols.append(Symbol(
                id=method_id, type="method", name=item.name,
                file=rel_path, language="python",
                line_start=item.lineno,
                line_end=item.end_lineno or item.lineno,
                scope=node.name, params=params, decorators=m_decorators,
                docstring=mdoc[:200] if mdoc else None,
                exported=not item.name.startswith("_"),
            ))
            edges.append(Edge(sym_id, method_id, "contains"))


def _extract_python_function(
    node, rel_path: str, file_id: str, symbols: list, edges: list,
):
    func_id = f"function:{rel_path}:{node.name}"
    params = [a.arg for a in node.args.args]
    docstring = python_ast.get_docstring(node)
    decorators = []
    for dec in node.decorator_list:
        try:
            decorators.append(python_ast.unparse(dec))
        except Exception:
            pass

    symbols.append(Symbol(
        id=func_id, type="function", name=node.name,
        file=rel_path, language="python",
        line_start=node.lineno,
        line_end=node.end_lineno or node.lineno,
        params=params, decorators=decorators,
        docstring=docstring[:200] if docstring else None,
        exported=not node.name.startswith("_"),
    ))
    edges.append(Edge(file_id, func_id, "contains"))


# ── Regex Parsers ─────────────────────────────────────────────────

# Regex patterns per language: list of (pattern, symbol_type, group_indices)
# Each pattern extracts: name (required), optional: params, bases, scope

GO_PATTERNS = [
    (r"^package\s+(\w+)", "package", {"name": 1}),
    (r'^func\s+(\w+)\s*\(([^)]*)\)', "function", {"name": 1, "params": 2}),
    (r'^func\s+\(\w+\s+\*?(\w+)\)\s+(\w+)\s*\(([^)]*)\)', "method", {"scope": 1, "name": 2, "params": 3}),
    (r'^type\s+(\w+)\s+struct\s*\{', "struct", {"name": 1}),
    (r'^type\s+(\w+)\s+interface\s*\{', "interface", {"name": 1}),
    (r'^\s*"([^"]+)"', "_import", {"name": 1}),  # inside import block
]

GO_IMPORT_BLOCK = re.compile(r'^import\s*\(\s*\n(.*?)\n\s*\)', re.MULTILINE | re.DOTALL)
GO_IMPORT_SINGLE = re.compile(r'^import\s+"([^"]+)"', re.MULTILINE)

JS_PATTERNS = [
    (r"^(?:export\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?", "class", {"name": 1, "bases": 2}),
    (r"^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)", "function", {"name": 1, "params": 2}),
    (r"^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:\([^)]*\)|[^=])\s*=>", "function", {"name": 1}),
    (r'^import\s+.*?\s+from\s+[\'"]([^\'"]+)[\'"]', "_import", {"name": 1}),
    (r'^import\s+[\'"]([^\'"]+)[\'"]', "_import", {"name": 1}),
    (r"require\(\s*['\"]([^'\"]+)['\"]\s*\)", "_import", {"name": 1}),
    (r"^(?:export\s+)?interface\s+(\w+)", "interface", {"name": 1}),
    (r"^(?:export\s+)?type\s+(\w+)\s*=", "interface", {"name": 1}),
    (r"^(?:export\s+)?enum\s+(\w+)", "enum", {"name": 1}),
]

JAVA_PATTERNS = [
    (r"^package\s+([\w.]+);", "package", {"name": 1}),
    (r"^import\s+([\w.]+);", "_import", {"name": 1}),
    (r"(?:public|private|protected)?\s*(?:abstract\s+)?(?:static\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?", "class", {"name": 1, "bases": 2}),
    (r"(?:public|private|protected)?\s*interface\s+(\w+)", "interface", {"name": 1}),
    (r"(?:public|private|protected)?\s*enum\s+(\w+)", "enum", {"name": 1}),
    (r"(?:public|private|protected)\s+(?:static\s+)?(?:abstract\s+)?(?:synchronized\s+)?(?:final\s+)?\w+(?:<[^>]+>)?\s+(\w+)\s*\(([^)]*)\)", "method", {"name": 1, "params": 2}),
]

RUST_PATTERNS = [
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
]

PROTO_PATTERNS = [
    (r'^message\s+(\w+)\s*\{', "class", {"name": 1}),
    (r'^service\s+(\w+)\s*\{', "interface", {"name": 1}),
    (r'^\s*rpc\s+(\w+)\s*\((\w+)\)\s*returns\s*\((\w+)\)', "method", {"name": 1, "params": 2}),
    (r'^enum\s+(\w+)\s*\{', "enum", {"name": 1}),
    (r'^import\s+"([^"]+)"', "_import", {"name": 1}),
]

CSHARP_PATTERNS = [
    (r"(?:public|private|protected|internal)?\s*(?:abstract\s+)?(?:static\s+)?class\s+(\w+)(?:\s*:\s*(\w+))?", "class", {"name": 1, "bases": 2}),
    (r"(?:public|private|protected|internal)?\s*interface\s+(\w+)", "interface", {"name": 1}),
    (r"(?:public|private|protected|internal)?\s*enum\s+(\w+)", "enum", {"name": 1}),
    (r"(?:public|private|protected|internal)\s+(?:static\s+)?(?:async\s+)?(?:virtual\s+)?(?:override\s+)?\w+(?:<[^>]+>)?\s+(\w+)\s*\(([^)]*)\)", "method", {"name": 1, "params": 2}),
    (r'^using\s+([\w.]+);', "_import", {"name": 1}),
]

LANG_PATTERNS = {
    "go": GO_PATTERNS,
    "javascript": JS_PATTERNS,
    "typescript": JS_PATTERNS,
    "java": JAVA_PATTERNS,
    "rust": RUST_PATTERNS,
    "protobuf": PROTO_PATTERNS,
    "csharp": CSHARP_PATTERNS,
}


def parse_with_regex(
    content: str, rel_path: str, language: str,
) -> tuple[list[Symbol], list[Edge]]:
    """Parse a source file using regex patterns."""
    symbols = []
    edges = []
    file_id = f"file:{rel_path}"
    patterns = LANG_PATTERNS.get(language, [])
    if not patterns:
        return symbols, edges

    lines = content.split("\n")

    # Handle Go import blocks specially
    if language == "go":
        for m in GO_IMPORT_BLOCK.finditer(content):
            block = m.group(1)
            for line in block.strip().split("\n"):
                line = line.strip().strip('"')
                if line and not line.startswith("//"):
                    # Handle aliased imports
                    parts = line.split()
                    imp = parts[-1].strip('"') if parts else line
                    edges.append(Edge(file_id, f"module:{imp}", "imports"))
        for m in GO_IMPORT_SINGLE.finditer(content):
            edges.append(Edge(file_id, f"module:{m.group(1)}", "imports"))

    current_class = None
    brace_depth = 0

    for lineno, line in enumerate(lines, 1):
        stripped = line.rstrip()

        # Track brace depth for scope detection
        brace_depth += stripped.count("{") - stripped.count("}")
        if brace_depth <= 0:
            current_class = None

        for pattern_str, sym_type, groups in patterns:
            m = re.search(pattern_str, stripped)
            if not m:
                continue

            name = m.group(groups["name"]) if "name" in groups else None
            if not name:
                continue

            if sym_type == "_import":
                if language != "go":  # Go imports handled above
                    edges.append(Edge(file_id, f"module:{name}", "imports"))
                continue

            if sym_type == "package":
                continue

            params_str = m.group(groups["params"]) if "params" in groups and groups["params"] <= len(m.groups()) else None
            params = [p.strip().split()[-1].split(":")[0] for p in params_str.split(",") if p.strip()] if params_str else []

            bases_match = m.group(groups["bases"]) if "bases" in groups and groups["bases"] <= len(m.groups()) and m.group(groups["bases"]) else None
            bases = [bases_match] if bases_match else []

            scope_match = m.group(groups["scope"]) if "scope" in groups and groups["scope"] <= len(m.groups()) and m.group(groups["scope"]) else None

            # Determine scope
            scope = scope_match or (current_class if sym_type == "method" else None)

            # Determine end line (rough: find next symbol or end of file)
            end_line = lineno
            for j in range(lineno, min(lineno + 500, len(lines))):
                if j > lineno and lines[j - 1].strip() and not lines[j - 1].startswith((" ", "\t")) and brace_depth <= 1:
                    end_line = j - 1
                    break
            else:
                end_line = min(lineno + 50, len(lines))

            # Build symbol ID
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

            break  # Only match first pattern per line

    return symbols, edges


# ── Tree-sitter Integration ──────────────────────────────────────

def try_load_tree_sitter() -> dict:
    """Try to load tree-sitter with available language grammars."""
    available = {}

    try:
        from tree_sitter import Language, Parser
    except ImportError:
        return available

    # Try new API (>= 0.22): individual grammar packages
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
        except Exception:
            pass

    # Try bundled grammars package
    if not available:
        try:
            from tree_sitter_languages import get_parser, get_language
            for lang in ["python", "go", "javascript", "typescript", "java", "rust", "ruby", "c", "cpp"]:
                try:
                    available[lang] = (get_parser(lang), get_language(lang))
                except Exception:
                    pass
        except ImportError:
            pass

    return available


# Tree-sitter node types to extract per language
TS_NODE_TYPES = {
    "python": {
        "class": ["class_definition"],
        "function": ["function_definition"],
    },
    "go": {
        "function": ["function_declaration"],
        "method": ["method_declaration"],
        "struct": ["type_spec"],
        "interface": ["type_spec"],
    },
    "javascript": {
        "class": ["class_declaration"],
        "function": ["function_declaration", "arrow_function"],
        "interface": ["interface_declaration"],
    },
    "typescript": {
        "class": ["class_declaration"],
        "function": ["function_declaration", "arrow_function"],
        "interface": ["interface_declaration", "type_alias_declaration"],
    },
    "java": {
        "class": ["class_declaration"],
        "interface": ["interface_declaration"],
        "enum": ["enum_declaration"],
        "method": ["method_declaration", "constructor_declaration"],
    },
    "rust": {
        "function": ["function_item"],
        "struct": ["struct_item"],
        "enum": ["enum_item"],
        "interface": ["trait_item"],
    },
}


def parse_with_tree_sitter(
    content: str, rel_path: str, language: str, parser, lang_obj,
) -> tuple[list[Symbol], list[Edge]]:
    """Parse a file using tree-sitter for more accurate symbol extraction."""
    symbols = []
    edges = []
    file_id = f"file:{rel_path}"

    try:
        tree = parser.parse(content.encode("utf-8"))
    except Exception:
        return symbols, edges

    node_types = TS_NODE_TYPES.get(language, {})
    all_types = set()
    for types in node_types.values():
        all_types.update(types)

    def visit(node, scope=None):
        node_type = node.type
        if node_type in all_types:
            # Find which symbol type this maps to
            sym_type = None
            for st, types in node_types.items():
                if node_type in types:
                    sym_type = st
                    break

            if sym_type:
                # Extract name from first identifier child
                name = None
                for child in node.children:
                    if child.type in ("identifier", "name", "type_identifier"):
                        name = content[child.start_byte:child.end_byte]
                        break

                if name:
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
                        line_start=node.start_point[0] + 1,
                        line_end=node.end_point[0] + 1,
                        scope=scope, exported=exported,
                    ))

                    if sym_type in ("class", "struct", "interface", "enum"):
                        edges.append(Edge(file_id, sym_id, "contains"))
                        # Recurse into class body with scope
                        for child in node.children:
                            visit(child, scope=name)
                        return
                    elif sym_type == "method" and scope:
                        edges.append(Edge(f"class:{rel_path}:{scope}", sym_id, "contains"))
                    else:
                        edges.append(Edge(file_id, sym_id, "contains"))

        for child in node.children:
            visit(child, scope)

    visit(tree.root_node)

    # Still use regex for imports (more reliable across languages)
    _, import_edges = parse_with_regex(content, rel_path, language)
    import_edges = [e for e in import_edges if e.type == "imports"]
    edges.extend(import_edges)

    return symbols, edges


# ── Parser Dispatcher ─────────────────────────────────────────────

def parse_file(
    abs_path: str, rel_path: str, language: str,
    ts_parsers: dict,
) -> tuple[list[Symbol], list[Edge]]:
    """Parse a single file, choosing the best available parser."""
    content = read_file(abs_path)
    if content is None:
        return [], []

    # Python: always use ast module (most accurate)
    if language == "python":
        return parse_python(content, rel_path)

    # Tree-sitter if available for this language
    if language in ts_parsers:
        parser, lang_obj = ts_parsers[language]
        symbols, edges = parse_with_tree_sitter(content, rel_path, language, parser, lang_obj)
        if symbols or edges:
            return symbols, edges

    # Fallback: regex
    return parse_with_regex(content, rel_path, language)


# ── Graph Builder ─────────────────────────────────────────────────

def resolve_imports(
    symbols: list[Symbol], edges: list[Edge], file_map: dict[str, str],
) -> list[Edge]:
    """Resolve import edges to actual files where possible."""
    resolved = []

    # Build lookup maps
    module_to_files = defaultdict(list)
    for rel_path, lang in file_map.items():
        # Convert file path to module-like name
        parts = Path(rel_path).with_suffix("").parts
        module_name = ".".join(parts)
        module_to_files[module_name].append(rel_path)

        # Also map partial paths
        for i in range(len(parts)):
            partial = ".".join(parts[i:])
            module_to_files[partial].append(rel_path)

    # Map Go import paths
    go_pkg_map = defaultdict(list)
    for rel_path, lang in file_map.items():
        if lang == "go":
            pkg_dir = str(Path(rel_path).parent)
            go_pkg_map[pkg_dir].append(rel_path)

    symbol_map = {}
    for sym in symbols:
        symbol_map[sym.name] = sym
        if sym.scope:
            symbol_map[f"{sym.scope}.{sym.name}"] = sym

    for edge in edges:
        if edge.type != "imports":
            resolved.append(edge)
            continue

        target = edge.target
        if target.startswith("module:"):
            module_name = target[7:]

            # Try direct file mapping
            candidates = module_to_files.get(module_name, [])
            if candidates:
                for c in candidates:
                    resolved.append(Edge(edge.source, f"file:{c}", "imports", edge.metadata))
                continue

            # Try Go package mapping
            for pkg_path, files in go_pkg_map.items():
                if module_name.endswith(pkg_path.replace("/", ".")):
                    for f in files:
                        resolved.append(Edge(edge.source, f"file:{f}", "imports", edge.metadata))
                    break
            else:
                # Keep unresolved (external dependency)
                resolved.append(edge)

        elif target.startswith("symbol:"):
            symbol_path = target[7:]
            parts = symbol_path.rsplit(".", 1)
            if len(parts) == 2:
                module_part, sym_name = parts
                candidates = module_to_files.get(module_part, [])
                if candidates:
                    for c in candidates:
                        # Find the actual symbol
                        for sym in symbols:
                            if sym.file == c and sym.name == sym_name:
                                resolved.append(Edge(edge.source, sym.id, "imports", edge.metadata))
                                break
                        else:
                            resolved.append(Edge(edge.source, f"file:{c}", "imports", edge.metadata))
                    continue
            resolved.append(edge)
        else:
            resolved.append(edge)

    # Resolve inheritance edges
    final = []
    for edge in resolved:
        if edge.type == "inherits" and edge.metadata.get("unresolved"):
            base_name = edge.target.split(":")[-1]
            # Search for the base class
            found = False
            for sym in symbols:
                if sym.name == base_name and sym.type in ("class", "struct", "interface"):
                    final.append(Edge(edge.source, sym.id, "inherits"))
                    found = True
                    break
            if not found:
                final.append(edge)  # Keep unresolved
        else:
            final.append(edge)

    return final


def compute_directory_stats(
    files: list[tuple[str, str, str]], symbols: list[Symbol],
) -> dict:
    """Compute statistics per directory."""
    dir_stats = defaultdict(lambda: {"files": 0, "languages": set(), "symbols": 0, "classes": 0, "functions": 0})

    for _, rel_path, lang in files:
        d = str(Path(rel_path).parent) or "."
        dir_stats[d]["files"] += 1
        dir_stats[d]["languages"].add(lang)

    for sym in symbols:
        d = str(Path(sym.file).parent) or "."
        dir_stats[d]["symbols"] += 1
        if sym.type in ("class", "struct", "interface"):
            dir_stats[d]["classes"] += 1
        elif sym.type in ("function", "method"):
            dir_stats[d]["functions"] += 1

    # Convert sets to lists for JSON
    for d in dir_stats:
        dir_stats[d]["languages"] = sorted(dir_stats[d]["languages"])

    return dict(dir_stats)


# ── Output Generators ─────────────────────────────────────────────

def generate_graph_json(
    symbols: list[Symbol], edges: list[Edge],
    files: list[tuple[str, str, str]], repo_root: str,
) -> dict:
    """Generate the full knowledge graph as a JSON-serializable dict."""
    lang_counts = defaultdict(int)
    for _, _, lang in files:
        lang_counts[lang] += 1

    type_counts = defaultdict(int)
    for sym in symbols:
        type_counts[sym.type] += 1

    # Build file nodes
    nodes = []
    for abs_path, rel_path, lang in files:
        try:
            line_count = sum(1 for _ in open(abs_path, "r", encoding="utf-8", errors="ignore"))
        except OSError:
            line_count = 0

        nodes.append({
            "id": f"file:{rel_path}",
            "type": "file",
            "name": Path(rel_path).name,
            "path": rel_path,
            "language": lang,
            "line_count": line_count,
        })

    # Add symbol nodes
    for sym in symbols:
        node = asdict(sym)
        # Remove None values
        node = {k: v for k, v in node.items() if v is not None and v != [] and v != False}
        if "exported" not in node:
            node["exported"] = False
        nodes.append(node)

    # Serialize edges
    edge_list = [asdict(e) for e in edges]
    # Remove empty metadata
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
        if not sym.exported:
            tag["private"] = True
        tags.append(tag)
    return sorted(tags, key=lambda t: (t["file"], t["line"]))


def generate_modules_json(edges: list[Edge], files: list[tuple[str, str, str]]) -> dict:
    """Generate module-level dependency map."""
    # Group files by directory (module)
    dir_files = defaultdict(list)
    for _, rel_path, lang in files:
        d = str(Path(rel_path).parent) or "."
        dir_files[d].append(rel_path)

    # Map files to their directory
    file_to_dir = {}
    for _, rel_path, _ in files:
        file_to_dir[rel_path] = str(Path(rel_path).parent) or "."

    # Build module dependency edges
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

    return {
        "modules": {
            d: {
                "files": sorted(f),
                "depends_on": sorted(module_deps.get(d, set())),
            }
            for d, f in sorted(dir_files.items())
        }
    }


def generate_summary_md(
    symbols: list[Symbol], edges: list[Edge],
    files: list[tuple[str, str, str]], dir_stats: dict,
    repo_root: str, ts_available: bool,
) -> str:
    """Generate a human/AI-readable codebase summary."""
    lang_counts = defaultdict(int)
    for _, _, lang in files:
        lang_counts[lang] += 1

    type_counts = defaultdict(int)
    for sym in symbols:
        type_counts[sym.type] += 1

    # Find hub files (most imported)
    import_counts = defaultdict(int)
    for edge in edges:
        if edge.type == "imports" and edge.target.startswith("file:"):
            import_counts[edge.target[5:]] += 1
    hub_files = sorted(import_counts.items(), key=lambda x: -x[1])[:15]

    # Find dependency roots (files that import nothing internal)
    file_imports = defaultdict(set)
    for edge in edges:
        if edge.type == "imports" and edge.source.startswith("file:") and edge.target.startswith("file:"):
            file_imports[edge.source[5:]].add(edge.target[5:])

    # Build inheritance trees
    inheritance = defaultdict(list)
    for edge in edges:
        if edge.type == "inherits" and not edge.metadata.get("unresolved"):
            child_name = edge.source.split(":")[-1]
            parent_name = edge.target.split(":")[-1]
            inheritance[parent_name].append(child_name)

    lines = []
    lines.append("# Codebase Knowledge Graph\n")
    lines.append(f"Generated: {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M UTC')}")
    lines.append(f"Parser: {'Tree-sitter + AST' if ts_available else 'AST + Regex'}\n")

    # Statistics
    lines.append("## Statistics\n")
    lines.append(f"- **{len(files)}** files across **{len(lang_counts)}** languages")
    lines.append(f"- **{len(symbols)}** symbols extracted")
    for st, count in sorted(type_counts.items(), key=lambda x: -x[1]):
        plural = st if st.endswith("s") else f"{st}es" if st.endswith("ss") else f"{st}s"
        lines.append(f"  - {count} {plural if count != 1 else st}")
    lines.append(f"- **{sum(1 for e in edges if e.type == 'imports')}** import relationships")
    lines.append(f"- **{sum(1 for e in edges if e.type == 'inherits')}** inheritance relationships")
    lines.append("")

    # Language breakdown
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
        stats = dir_stats[d]
        indent = "  " * d.count("/")
        dir_name = Path(d).name or "."
        lang_str = ", ".join(stats["languages"])
        lines.append(f"{indent}{dir_name}/ ({stats['files']} files, {lang_str})")
    lines.append("```\n")

    # Hub files
    if hub_files:
        lines.append("## Hub Files (most imported)\n")
        lines.append("| File | Imported by |")
        lines.append("|------|-------------|")
        for f, count in hub_files:
            lines.append(f"| `{f}` | {count} files |")
        lines.append("")

    # Key classes
    classes = [s for s in symbols if s.type in ("class", "struct", "interface") and s.exported]
    if classes:
        lines.append("## Key Types\n")
        lines.append("| Type | Kind | File | Line | Bases |")
        lines.append("|------|------|------|------|-------|")
        for cls in sorted(classes, key=lambda c: c.name)[:50]:
            bases_str = ", ".join(cls.bases) if cls.bases else "-"
            doc_str = f" - {cls.docstring[:80]}" if cls.docstring else ""
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
    exported_funcs = [s for s in symbols if s.type == "function" and s.exported]
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

    # Module dependencies
    file_set = {rel for _, rel, _ in files}
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

    return "\n".join(lines)


# ── Main ──────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Build a knowledge graph of a codebase",
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
        "--include-tests", action="store_true",
        help="Include test files (excluded by default)",
    )
    parser.add_argument(
        "--no-tree-sitter", action="store_true",
        help="Disable tree-sitter even if available",
    )
    parser.add_argument(
        "--quiet", "-q", action="store_true",
        help="Suppress progress output",
    )

    args = parser.parse_args()
    repo_root = os.path.abspath(args.repo_root)
    output_dir = args.output_dir or os.path.join(repo_root, "docs", "code-tree")
    excludes = DEFAULT_EXCLUDES + args.exclude

    def log(msg):
        if not args.quiet:
            print(msg, file=sys.stderr)

    log(f"Scanning: {repo_root}")

    # Load tree-sitter
    ts_parsers = {}
    if not args.no_tree_sitter:
        ts_parsers = try_load_tree_sitter()
        if ts_parsers:
            log(f"Tree-sitter available for: {', '.join(sorted(ts_parsers.keys()))}")
        else:
            log("Tree-sitter not available, using AST/regex parsers")

    # Discover files
    files = discover_files(repo_root, excludes, args.include_tests, args.max_file_size)
    log(f"Found {len(files)} source files")

    if not files:
        log("No files found. Check --repo-root and --exclude options.")
        sys.exit(1)

    # Parse all files
    all_symbols = []
    all_edges = []
    file_map = {}  # rel_path -> language

    for i, (abs_path, rel_path, lang) in enumerate(files):
        if not args.quiet and (i + 1) % 100 == 0:
            log(f"  Parsed {i + 1}/{len(files)} files...")

        file_map[rel_path] = lang
        symbols, edges = parse_file(abs_path, rel_path, lang, ts_parsers)
        all_symbols.extend(symbols)
        all_edges.extend(edges)

    log(f"Extracted {len(all_symbols)} symbols, {len(all_edges)} edges")

    # Resolve imports
    all_edges = resolve_imports(all_symbols, all_edges, file_map)
    log(f"Resolved imports: {len(all_edges)} edges after resolution")

    # Compute directory stats
    dir_stats = compute_directory_stats(files, all_symbols)

    # Generate outputs
    os.makedirs(output_dir, exist_ok=True)

    # graph.json
    graph = generate_graph_json(all_symbols, all_edges, files, repo_root)
    graph_path = os.path.join(output_dir, "graph.json")
    with open(graph_path, "w") as f:
        json.dump(graph, f, indent=2, default=str)
    log(f"Wrote {graph_path}")

    # tags.json
    tags = generate_tags_json(all_symbols)
    tags_path = os.path.join(output_dir, "tags.json")
    with open(tags_path, "w") as f:
        json.dump(tags, f, indent=2)
    log(f"Wrote {tags_path}")

    # modules.json
    modules = generate_modules_json(all_edges, files)
    modules_path = os.path.join(output_dir, "modules.json")
    with open(modules_path, "w") as f:
        json.dump(modules, f, indent=2)
    log(f"Wrote {modules_path}")

    # summary.md
    summary = generate_summary_md(
        all_symbols, all_edges, files, dir_stats,
        repo_root, bool(ts_parsers),
    )
    summary_path = os.path.join(output_dir, "summary.md")
    with open(summary_path, "w") as f:
        f.write(summary)
    log(f"Wrote {summary_path}")

    log(f"\nDone. Output in: {output_dir}")
    log(f"  graph.json   - Full knowledge graph ({len(graph['nodes'])} nodes, {len(graph['edges'])} edges)")
    log(f"  tags.json    - Symbol index ({len(tags)} tags)")
    log(f"  modules.json - Module dependencies ({len(modules['modules'])} modules)")
    log(f"  summary.md   - Codebase overview")


if __name__ == "__main__":
    main()
