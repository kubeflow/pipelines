#!/usr/bin/env python3
"""Unit tests for code_tree.py — covers parsing, call resolution, import
resolution, test mapping, incremental mode, and output generation."""

import textwrap
import unittest
from collections import defaultdict

from code_tree import (
    CallRef,
    Edge,
    ModuleBoundary,
    Symbol,
    _edge_file,
    _extract_go_import_edges,
    _extract_regex_imports,
    _match_test_symbol_to_source,
    _match_test_to_source_path,
    compute_directory_stats,
    compute_graph_stats,
    detect_test_relationships,
    generate_tags_json,
    is_test_file,
    parse_python,
    parse_with_regex,
    resolve_calls,
    resolve_imports,
    should_exclude,
)


class TestRelativeImportResolution(unittest.TestCase):
    """Fix 1: Relative JS/TS and Python imports must resolve to local files."""

    def test_js_relative_import_sibling(self):
        """import from './utils' resolves to src/utils.ts."""
        symbols = []
        edges = [
            Edge("file:src/app.ts", "module:./utils", "imports"),
        ]
        file_map = {"src/app.ts": "typescript", "src/utils.ts": "typescript"}
        result = resolve_imports(symbols, edges, file_map, [])
        resolved_targets = [e.target for e in result if e.type == "imports"]
        self.assertIn("file:src/utils.ts", resolved_targets)
        self.assertNotIn("module:./utils", [e.target for e in result])

    def test_js_relative_import_parent_dir(self):
        """import from '../providers/utils' resolves correctly."""
        symbols = []
        edges = [
            Edge("file:src/components/app.tsx", "module:../providers/utils", "imports"),
        ]
        file_map = {
            "src/components/app.tsx": "typescript",
            "src/providers/utils.ts": "typescript",
        }
        result = resolve_imports(symbols, edges, file_map, [])
        resolved_targets = [e.target for e in result if e.type == "imports"]
        self.assertIn("file:src/providers/utils.ts", resolved_targets)

    def test_js_relative_import_index_file(self):
        """import from './components' resolves to components/index.ts."""
        symbols = []
        edges = [
            Edge("file:src/app.ts", "module:./components", "imports"),
        ]
        file_map = {
            "src/app.ts": "typescript",
            "src/components/index.ts": "typescript",
        }
        result = resolve_imports(symbols, edges, file_map, [])
        resolved_targets = [e.target for e in result if e.type == "imports"]
        self.assertIn("file:src/components/index.ts", resolved_targets)

    def test_js_relative_import_jsx_extension(self):
        """import from './Button' resolves to Button.jsx."""
        symbols = []
        edges = [
            Edge("file:src/app.js", "module:./Button", "imports"),
        ]
        file_map = {"src/app.js": "javascript", "src/Button.jsx": "javascript"}
        result = resolve_imports(symbols, edges, file_map, [])
        resolved_targets = [e.target for e in result if e.type == "imports"]
        self.assertIn("file:src/Button.jsx", resolved_targets)

    def test_python_relative_import(self):
        """Python relative import ./foo resolves to foo.py in same dir."""
        symbols = []
        edges = [
            Edge("file:pkg/bar.py", "module:./foo", "imports"),
        ]
        file_map = {"pkg/bar.py": "python", "pkg/foo.py": "python"}
        result = resolve_imports(symbols, edges, file_map, [])
        resolved_targets = [e.target for e in result if e.type == "imports"]
        self.assertIn("file:pkg/foo.py", resolved_targets)

    def test_python_relative_import_init(self):
        """Python relative import ./sub resolves to sub/__init__.py."""
        symbols = []
        edges = [
            Edge("file:pkg/bar.py", "module:./sub", "imports"),
        ]
        file_map = {"pkg/bar.py": "python", "pkg/sub/__init__.py": "python"}
        result = resolve_imports(symbols, edges, file_map, [])
        resolved_targets = [e.target for e in result if e.type == "imports"]
        self.assertIn("file:pkg/sub/__init__.py", resolved_targets)

    def test_unresolvable_relative_import_kept(self):
        """Relative import with no matching file stays as unresolved edge."""
        symbols = []
        edges = [
            Edge("file:src/app.ts", "module:./nonexistent", "imports"),
        ]
        file_map = {"src/app.ts": "typescript"}
        result = resolve_imports(symbols, edges, file_map, [])
        unresolved = [e for e in result if e.target == "module:./nonexistent"]
        self.assertEqual(len(unresolved), 1)

    def test_non_relative_import_unchanged(self):
        """Absolute module imports still go through the existing path."""
        symbols = []
        edges = [
            Edge("file:src/app.ts", "module:react", "imports"),
        ]
        file_map = {"src/app.ts": "typescript"}
        result = resolve_imports(symbols, edges, file_map, [])
        # 'react' is external, should remain as-is
        kept = [e for e in result if e.target == "module:react"]
        self.assertEqual(len(kept), 1)


class TestQualifiedTestMatching(unittest.TestCase):
    """Fix 2: test_Class_method must match Class.method via symbol_by_name."""

    def _make_symbols(self):
        """Create source and test symbols for method-level matching."""
        source_method = Symbol(
            id="method:src/a.py:MyClass.process",
            type="method", name="process",
            file="src/a.py", language="python",
            line_start=10, line_end=20,
            scope="MyClass", exported=True, is_test=False,
        )
        source_class = Symbol(
            id="class:src/a.py:MyClass",
            type="class", name="MyClass",
            file="src/a.py", language="python",
            line_start=5, line_end=25,
            exported=True, is_test=False,
        )
        test_func = Symbol(
            id="function:tests/test_a.py:test_MyClass_process",
            type="function", name="test_MyClass_process",
            file="tests/test_a.py", language="python",
            line_start=1, line_end=5,
            exported=True, is_test=True,
        )
        return source_class, source_method, test_func

    def test_qualified_name_in_index(self):
        """symbol_by_name must contain 'MyClass.process' key."""
        source_class, source_method, test_func = self._make_symbols()
        all_symbols = [source_class, source_method, test_func]
        files = [
            ("/abs/src/a.py", "src/a.py", "python", False),
            ("/abs/tests/test_a.py", "tests/test_a.py", "python", True),
        ]
        file_map = {"src/a.py": "python", "tests/test_a.py": "python"}

        test_edges = detect_test_relationships(all_symbols, [], files, file_map)
        # Should have a symbol-level tests edge from test_func to source_method
        symbol_test_edges = [
            e for e in test_edges
            if e.source == test_func.id and e.target == source_method.id
        ]
        self.assertEqual(len(symbol_test_edges), 1,
                         "test_MyClass_process should create a tests edge to MyClass.process")

    def test_bare_name_still_works(self):
        """test_foo should still match function foo (bare name matching)."""
        source_func = Symbol(
            id="function:src/a.py:foo",
            type="function", name="foo",
            file="src/a.py", language="python",
            line_start=1, line_end=5,
            exported=True, is_test=False,
        )
        test_func = Symbol(
            id="function:tests/test_a.py:test_foo",
            type="function", name="test_foo",
            file="tests/test_a.py", language="python",
            line_start=1, line_end=5,
            exported=True, is_test=True,
        )
        # Build the index the same way detect_test_relationships does
        symbol_by_name = defaultdict(list)
        for sym in [source_func, test_func]:
            if not sym.is_test:
                symbol_by_name[sym.name].append(sym)
                if sym.scope:
                    symbol_by_name[f"{sym.scope}.{sym.name}"].append(sym)

        matches = _match_test_symbol_to_source(test_func, symbol_by_name)
        self.assertEqual(len(matches), 1)
        self.assertEqual(matches[0].id, source_func.id)

    def test_test_class_prefix_still_works(self):
        """TestFoo should match class Foo."""
        source_class = Symbol(
            id="class:src/a.py:Foo",
            type="class", name="Foo",
            file="src/a.py", language="python",
            line_start=1, line_end=10,
            exported=True, is_test=False,
        )
        test_class = Symbol(
            id="class:tests/test_a.py:TestFoo",
            type="class", name="TestFoo",
            file="tests/test_a.py", language="python",
            line_start=1, line_end=10,
            exported=True, is_test=True,
        )
        symbol_by_name = defaultdict(list)
        for sym in [source_class]:
            symbol_by_name[sym.name].append(sym)

        matches = _match_test_symbol_to_source(test_class, symbol_by_name)
        self.assertEqual(len(matches), 1)
        self.assertEqual(matches[0].id, source_class.id)


class TestIncrementalEdgePreservation(unittest.TestCase):
    """Fix 3: Incremental mode must keep edges from unchanged files
    even when they point at changed files."""

    def _simulate_incremental_merge(self, prev_edges, changed_files):
        """Simulate the incremental edge merge logic from main().

        This mirrors the fixed logic: keep edges whose source is from
        an unchanged file.
        """
        from code_tree import _edge_file

        kept = []
        for edge_data in prev_edges:
            src_file = _edge_file(edge_data.get("source", ""))
            if src_file not in changed_files:
                kept.append(Edge(
                    source=edge_data["source"],
                    target=edge_data["target"],
                    type=edge_data["type"],
                    metadata=edge_data.get("metadata", {}),
                ))
        return kept

    def test_unchanged_to_changed_edge_preserved(self):
        """b.py (unchanged) imports a.py (changed) — edge must survive."""
        prev_edges = [
            {"source": "file:b.py", "target": "file:a.py", "type": "imports"},
        ]
        changed_files = {"a.py"}

        kept = self._simulate_incremental_merge(prev_edges, changed_files)
        self.assertEqual(len(kept), 1)
        self.assertEqual(kept[0].source, "file:b.py")
        self.assertEqual(kept[0].target, "file:a.py")

    def test_changed_to_unchanged_edge_dropped(self):
        """a.py (changed) imports b.py (unchanged) — edge dropped (reparsed)."""
        prev_edges = [
            {"source": "file:a.py", "target": "file:b.py", "type": "imports"},
        ]
        changed_files = {"a.py"}

        kept = self._simulate_incremental_merge(prev_edges, changed_files)
        self.assertEqual(len(kept), 0,
                         "Edges from changed files should be dropped (they get re-emitted by reparsing)")

    def test_unchanged_to_unchanged_edge_preserved(self):
        """b.py (unchanged) imports c.py (unchanged) — edge must survive."""
        prev_edges = [
            {"source": "file:b.py", "target": "file:c.py", "type": "imports"},
        ]
        changed_files = {"a.py"}

        kept = self._simulate_incremental_merge(prev_edges, changed_files)
        self.assertEqual(len(kept), 1)

    def test_symbol_edge_unchanged_source(self):
        """Symbol-level call from unchanged file to changed file is preserved."""
        prev_edges = [
            {
                "source": "function:b.py:bar",
                "target": "function:a.py:foo",
                "type": "calls",
                "metadata": {"line": 8},
            },
        ]
        changed_files = {"a.py"}

        kept = self._simulate_incremental_merge(prev_edges, changed_files)
        self.assertEqual(len(kept), 1)
        self.assertEqual(kept[0].type, "calls")

    def test_mixed_edges_correct_filtering(self):
        """Multiple edges with mixed changed/unchanged sources."""
        prev_edges = [
            {"source": "file:b.py", "target": "file:a.py", "type": "imports"},      # keep (source unchanged)
            {"source": "file:a.py", "target": "file:b.py", "type": "imports"},      # drop (source changed)
            {"source": "file:c.py", "target": "file:a.py", "type": "imports"},      # keep (source unchanged)
            {"source": "function:a.py:foo", "target": "function:b.py:bar", "type": "calls"},  # drop
            {"source": "function:b.py:bar", "target": "function:a.py:foo", "type": "calls"},  # keep
        ]
        changed_files = {"a.py"}

        kept = self._simulate_incremental_merge(prev_edges, changed_files)
        self.assertEqual(len(kept), 3)
        sources = [e.source for e in kept]
        self.assertTrue(all("a.py" not in s.split(":")[1] for s in sources if ":" in s),
                        "No kept edge should have a changed-file source")


class TestShouldExclude(unittest.TestCase):
    """Tests for should_exclude path filtering."""

    def test_exact_dir_match(self):
        self.assertTrue(should_exclude("node_modules/foo.js", ["node_modules"]))

    def test_nested_excluded_dir(self):
        self.assertTrue(should_exclude("src/vendor/lib.go", ["vendor"]))

    def test_non_excluded_path(self):
        self.assertFalse(should_exclude("src/main.py", ["node_modules", ".git"]))

    def test_glob_pattern(self):
        self.assertTrue(should_exclude("build/output.js", ["build"]))

    def test_deep_path_not_matching(self):
        self.assertFalse(should_exclude("src/app/components/Button.tsx", ["node_modules"]))


class TestIsTestFile(unittest.TestCase):
    """Tests for is_test_file detection."""

    def test_test_prefix(self):
        self.assertTrue(is_test_file("test_foo.py"))

    def test_test_suffix(self):
        self.assertTrue(is_test_file("foo_test.go"))

    def test_spec_suffix(self):
        self.assertTrue(is_test_file("Button.spec.tsx"))

    def test_tests_directory(self):
        self.assertTrue(is_test_file("tests/test_utils.py"))

    def test_underscore_tests_dir(self):
        self.assertTrue(is_test_file("__tests__/Button.test.js"))

    def test_not_test(self):
        self.assertFalse(is_test_file("src/utils.py"))

    def test_not_test_similar_name(self):
        self.assertFalse(is_test_file("src/testing_utils.py"))


class TestEdgeFile(unittest.TestCase):
    """Tests for _edge_file helper."""

    def test_file_prefix(self):
        self.assertEqual(_edge_file("file:src/a.py"), "src/a.py")

    def test_symbol_id(self):
        self.assertEqual(_edge_file("function:src/a.py:foo"), "src/a.py")

    def test_method_with_scope(self):
        self.assertEqual(_edge_file("method:src/b.py:MyClass.bar"), "src/b.py")

    def test_empty_string(self):
        self.assertEqual(_edge_file(""), "")

    def test_no_colon(self):
        self.assertEqual(_edge_file("bare"), "")


class TestParsePython(unittest.TestCase):
    """Tests for Python AST parsing."""

    def test_function_extraction(self):
        code = textwrap.dedent("""\
            def hello(name: str) -> str:
                return f"Hello {name}"
        """)
        symbols, edges, calls = parse_python(code, "app.py", False)
        func_syms = [s for s in symbols if s.type == "function"]
        self.assertEqual(len(func_syms), 1)
        self.assertEqual(func_syms[0].name, "hello")
        self.assertEqual(func_syms[0].params, ["name"])
        self.assertIn("-> str", func_syms[0].signature)

    def test_class_with_methods(self):
        code = textwrap.dedent("""\
            class Calculator:
                \"\"\"A simple calculator.\"\"\"
                def add(self, a, b):
                    return a + b

                def subtract(self, a, b):
                    return a - b
        """)
        symbols, edges, calls = parse_python(code, "calc.py", False)
        class_syms = [s for s in symbols if s.type == "class"]
        method_syms = [s for s in symbols if s.type == "method"]
        self.assertEqual(len(class_syms), 1)
        self.assertEqual(class_syms[0].name, "Calculator")
        self.assertEqual(class_syms[0].docstring, "A simple calculator.")
        self.assertEqual(len(method_syms), 2)
        method_names = {m.name for m in method_syms}
        self.assertEqual(method_names, {"add", "subtract"})
        # self should be excluded from params
        for m in method_syms:
            self.assertNotIn("self", m.params)

    def test_class_inheritance(self):
        code = textwrap.dedent("""\
            class Child(Parent):
                pass
        """)
        symbols, edges, calls = parse_python(code, "child.py", False)
        inherits = [e for e in edges if e.type == "inherits"]
        self.assertEqual(len(inherits), 1)
        self.assertIn("Parent", inherits[0].target)

    def test_import_edges(self):
        code = textwrap.dedent("""\
            import os
            from pathlib import Path
            from . import utils
        """)
        symbols, edges, calls = parse_python(code, "pkg/main.py", False)
        import_edges = [e for e in edges if e.type == "imports"]
        targets = {e.target for e in import_edges}
        self.assertIn("module:os", targets)
        self.assertIn("symbol:pathlib.Path", targets)

    def test_call_extraction(self):
        code = textwrap.dedent("""\
            def process():
                result = compute(data)
                result.save()
        """)
        symbols, edges, calls = parse_python(code, "app.py", False)
        call_names = {c.raw_name for c in calls}
        self.assertIn("compute", call_names)
        self.assertIn("result.save", call_names)

    def test_builtin_calls_excluded(self):
        code = textwrap.dedent("""\
            def process():
                x = len(data)
                print(x)
                items = list(range(10))
        """)
        symbols, edges, calls = parse_python(code, "app.py", False)
        call_names = {c.raw_name for c in calls}
        self.assertNotIn("len", call_names)
        self.assertNotIn("print", call_names)
        self.assertNotIn("list", call_names)
        self.assertNotIn("range", call_names)

    def test_nested_function_calls_separate(self):
        """Nested functions should have their own caller_id."""
        code = textwrap.dedent("""\
            def outer():
                do_outer()
                def inner():
                    do_inner()
        """)
        symbols, edges, calls = parse_python(code, "app.py", False)
        outer_calls = [c for c in calls if "outer" in c.caller_id and "inner" not in c.caller_id]
        inner_calls = [c for c in calls if "inner" in c.caller_id]
        self.assertTrue(any(c.raw_name == "do_outer" for c in outer_calls))
        self.assertTrue(any(c.raw_name == "do_inner" for c in inner_calls))

    def test_constant_extraction(self):
        code = textwrap.dedent("""\
            MAX_SIZE = 1024
            _PRIVATE = "hidden"
            regular_var = 42
        """)
        symbols, edges, calls = parse_python(code, "config.py", False)
        constants = [s for s in symbols if s.type == "constant"]
        names = {c.name for c in constants}
        self.assertIn("MAX_SIZE", names)
        self.assertIn("_PRIVATE", names)
        self.assertNotIn("regular_var", names)

    def test_decorated_function(self):
        code = textwrap.dedent("""\
            @staticmethod
            def create():
                pass
        """)
        symbols, edges, calls = parse_python(code, "app.py", False)
        func = [s for s in symbols if s.type == "function"][0]
        self.assertIn("staticmethod", func.decorators)

    def test_test_function_flagged(self):
        code = textwrap.dedent("""\
            def test_something():
                assert True
        """)
        symbols, edges, calls = parse_python(code, "test_app.py", True)
        func = [s for s in symbols if s.type == "function"][0]
        self.assertTrue(func.is_test)

    def test_syntax_error_returns_empty(self):
        code = "def broken(:\n    pass"
        symbols, edges, calls = parse_python(code, "bad.py", False)
        self.assertEqual(len(symbols), 0)

    def test_async_function(self):
        code = textwrap.dedent("""\
            async def fetch(url):
                return await get(url)
        """)
        symbols, edges, calls = parse_python(code, "api.py", False)
        func_syms = [s for s in symbols if s.type == "function"]
        self.assertEqual(len(func_syms), 1)
        self.assertEqual(func_syms[0].name, "fetch")


class TestParseWithRegex(unittest.TestCase):
    """Tests for regex-based parsing (Go, JS, Java)."""

    def test_go_function(self):
        code = "func HandleRequest(w http.ResponseWriter, r *http.Request) {\n}\n"
        symbols, edges, calls = parse_with_regex(code, "handler.go", "go", False)
        func_syms = [s for s in symbols if s.type == "function"]
        self.assertEqual(len(func_syms), 1)
        self.assertEqual(func_syms[0].name, "HandleRequest")
        self.assertTrue(func_syms[0].exported)

    def test_go_unexported(self):
        code = "func handleInternal(ctx context.Context) {\n}\n"
        symbols, edges, calls = parse_with_regex(code, "handler.go", "go", False)
        func_syms = [s for s in symbols if s.type == "function"]
        self.assertEqual(len(func_syms), 1)
        self.assertFalse(func_syms[0].exported)

    def test_go_method(self):
        code = "func (s *Server) Start(port int) error {\n}\n"
        symbols, edges, calls = parse_with_regex(code, "server.go", "go", False)
        method_syms = [s for s in symbols if s.type == "method"]
        self.assertEqual(len(method_syms), 1)
        self.assertEqual(method_syms[0].name, "Start")
        self.assertEqual(method_syms[0].scope, "Server")

    def test_go_struct(self):
        code = "type Config struct {\n\tPort int\n}\n"
        symbols, edges, calls = parse_with_regex(code, "config.go", "go", False)
        struct_syms = [s for s in symbols if s.type == "struct"]
        self.assertEqual(len(struct_syms), 1)
        self.assertEqual(struct_syms[0].name, "Config")

    def test_go_interface(self):
        code = "type Handler interface {\n\tHandle(ctx context.Context) error\n}\n"
        symbols, edges, calls = parse_with_regex(code, "handler.go", "go", False)
        iface_syms = [s for s in symbols if s.type == "interface"]
        self.assertEqual(len(iface_syms), 1)
        self.assertEqual(iface_syms[0].name, "Handler")

    def test_js_class_with_extends(self):
        code = "export class Button extends Component {\n}\n"
        symbols, edges, calls = parse_with_regex(code, "Button.jsx", "javascript", False)
        class_syms = [s for s in symbols if s.type == "class"]
        self.assertEqual(len(class_syms), 1)
        self.assertEqual(class_syms[0].name, "Button")
        self.assertEqual(class_syms[0].bases, ["Component"])

    def test_js_arrow_function(self):
        code = "export const fetchData = async (url) => {\n}\n"
        symbols, edges, calls = parse_with_regex(code, "api.js", "javascript", False)
        func_syms = [s for s in symbols if s.type == "function"]
        self.assertEqual(len(func_syms), 1)
        self.assertEqual(func_syms[0].name, "fetchData")

    def test_js_import_edge(self):
        code = "import { useState } from 'react'\n"
        symbols, edges, calls = parse_with_regex(code, "app.jsx", "javascript", False)
        import_edges = [e for e in edges if e.type == "imports"]
        self.assertTrue(any(e.target == "module:react" for e in import_edges))

    def test_ts_interface(self):
        code = "export interface UserProps {\n  name: string;\n}\n"
        symbols, edges, calls = parse_with_regex(code, "types.ts", "typescript", False)
        iface_syms = [s for s in symbols if s.type == "interface"]
        self.assertEqual(len(iface_syms), 1)
        self.assertEqual(iface_syms[0].name, "UserProps")

    def test_ts_enum(self):
        code = "export enum Status {\n  Active,\n  Inactive\n}\n"
        symbols, edges, calls = parse_with_regex(code, "types.ts", "typescript", False)
        enum_syms = [s for s in symbols if s.type == "enum"]
        self.assertEqual(len(enum_syms), 1)
        self.assertEqual(enum_syms[0].name, "Status")

    def test_java_class(self):
        code = "public class UserService extends BaseService {\n}\n"
        symbols, edges, calls = parse_with_regex(code, "UserService.java", "java", False)
        class_syms = [s for s in symbols if s.type == "class"]
        self.assertEqual(len(class_syms), 1)
        self.assertEqual(class_syms[0].name, "UserService")
        self.assertEqual(class_syms[0].bases, ["BaseService"])

    def test_unsupported_language_returns_empty(self):
        symbols, edges, calls = parse_with_regex("some code", "file.lua", "lua", False)
        self.assertEqual(len(symbols), 0)
        self.assertEqual(len(edges), 0)


class TestExtractGoImportEdges(unittest.TestCase):
    """Tests for Go import block and single-line extraction."""

    def test_import_block(self):
        code = textwrap.dedent("""\
            import (
                "fmt"
                "net/http"
            )
        """)
        edges = _extract_go_import_edges(code, "file:main.go")
        targets = {e.target for e in edges}
        self.assertIn("module:fmt", targets)
        self.assertIn("module:net/http", targets)

    def test_single_import(self):
        code = 'import "os"\n'
        edges = _extract_go_import_edges(code, "file:main.go")
        self.assertTrue(any(e.target == "module:os" for e in edges))

    def test_aliased_import(self):
        code = textwrap.dedent("""\
            import (
                pb "google.golang.org/protobuf"
            )
        """)
        edges = _extract_go_import_edges(code, "file:main.go")
        self.assertTrue(any("protobuf" in e.target for e in edges))


class TestExtractRegexImports(unittest.TestCase):
    """Tests for _extract_regex_imports fallback."""

    def test_js_imports(self):
        code = "import { foo } from 'bar'\nimport 'side-effect'\n"
        edges = _extract_regex_imports(code, "app.js", "javascript")
        targets = {e.target for e in edges}
        self.assertIn("module:bar", targets)
        self.assertIn("module:side-effect", targets)

    def test_python_no_regex_imports(self):
        """Python imports are handled by ast, not regex. No patterns defined."""
        code = "import os\nfrom pathlib import Path\n"
        edges = _extract_regex_imports(code, "app.py", "python")
        self.assertEqual(len(edges), 0)

    def test_go_delegates_to_go_handler(self):
        code = 'import "fmt"\n'
        edges = _extract_regex_imports(code, "main.go", "go")
        self.assertTrue(any(e.target == "module:fmt" for e in edges))


class TestResolveCalls(unittest.TestCase):
    """Tests for call graph resolution."""

    def _make_symbols_and_edges(self):
        sym_foo = Symbol(
            id="function:a.py:foo", type="function", name="foo",
            file="a.py", language="python", line_start=1, line_end=5,
        )
        sym_bar = Symbol(
            id="function:b.py:bar", type="function", name="bar",
            file="b.py", language="python", line_start=1, line_end=10,
        )
        sym_cls = Symbol(
            id="class:a.py:MyClass", type="class", name="MyClass",
            file="a.py", language="python", line_start=10, line_end=30,
        )
        sym_method = Symbol(
            id="method:a.py:MyClass.process", type="method", name="process",
            file="a.py", language="python", line_start=15, line_end=25,
            scope="MyClass",
        )
        symbols = [sym_foo, sym_bar, sym_cls, sym_method]
        edges = [
            Edge("file:b.py", "file:a.py", "imports"),
            Edge("file:b.py", "symbol:a.foo", "imports", {"from_import": True}),
        ]
        return symbols, edges

    def test_direct_name_same_file(self):
        symbols, edges = self._make_symbols_and_edges()
        refs = [CallRef("function:a.py:foo", "process", 3, "a.py")]
        # process is a method in a.py, foo calls it by name
        call_edges = resolve_calls(refs, symbols, edges, {"a.py": "python"})
        # Should resolve "process" to method:a.py:MyClass.process
        targets = {e.target for e in call_edges}
        self.assertIn("method:a.py:MyClass.process", targets)

    def test_self_method_call_falls_back_to_name(self):
        """self.foo where foo isn't a method of MyClass falls back to name match."""
        symbols, edges = self._make_symbols_and_edges()
        refs = [CallRef("method:a.py:MyClass.process", "self.foo", 20, "a.py")]
        call_edges = resolve_calls(refs, symbols, edges, {"a.py": "python"})
        # foo is unique globally so last-resort name match resolves it
        self.assertEqual(len(call_edges), 1)
        self.assertEqual(call_edges[0].target, "function:a.py:foo")

    def test_imported_name_resolution(self):
        symbols, edges = self._make_symbols_and_edges()
        refs = [CallRef("function:b.py:bar", "foo", 5, "b.py")]
        call_edges = resolve_calls(refs, symbols, edges, {"b.py": "python"})
        targets = {e.target for e in call_edges}
        self.assertIn("function:a.py:foo", targets)

    def test_no_self_call(self):
        """A function should not resolve a call to itself."""
        symbols, edges = self._make_symbols_and_edges()
        refs = [CallRef("function:a.py:foo", "foo", 3, "a.py")]
        call_edges = resolve_calls(refs, symbols, edges, {"a.py": "python"})
        for e in call_edges:
            self.assertNotEqual(e.source, e.target)

    def test_deduplication(self):
        """Multiple calls to the same target from same caller produce one edge."""
        symbols, edges = self._make_symbols_and_edges()
        refs = [
            CallRef("function:b.py:bar", "foo", 5, "b.py"),
            CallRef("function:b.py:bar", "foo", 7, "b.py"),
        ]
        call_edges = resolve_calls(refs, symbols, edges, {"b.py": "python"})
        targets = [e.target for e in call_edges if e.source == "function:b.py:bar"]
        self.assertEqual(len(targets), 1)

    def test_qualified_class_method(self):
        """MyClass.process should resolve via class lookup in same file."""
        symbols, edges = self._make_symbols_and_edges()
        refs = [CallRef("function:a.py:foo", "MyClass.process", 3, "a.py")]
        call_edges = resolve_calls(refs, symbols, edges, {"a.py": "python"})
        targets = {e.target for e in call_edges}
        self.assertIn("method:a.py:MyClass.process", targets)


class TestMatchTestToSourcePath(unittest.TestCase):
    """Tests for test-to-source file path matching."""

    def test_test_prefix(self):
        source_files = {"src/utils.py"}
        result = _match_test_to_source_path("tests/test_utils.py", source_files)
        self.assertEqual(result, "src/utils.py")

    def test_test_suffix(self):
        source_files = {"handler.go"}
        result = _match_test_to_source_path("handler_test.go", source_files)
        self.assertEqual(result, "handler.go")

    def test_spec_suffix(self):
        source_files = {"src/Button.tsx"}
        result = _match_test_to_source_path("src/Button.spec.tsx", source_files)
        self.assertEqual(result, "src/Button.tsx")

    def test_same_directory(self):
        source_files = {"pkg/server.go"}
        result = _match_test_to_source_path("pkg/server_test.go", source_files)
        self.assertEqual(result, "pkg/server.go")

    def test_no_match(self):
        source_files = {"src/other.py"}
        result = _match_test_to_source_path("tests/test_utils.py", source_files)
        self.assertIsNone(result)

    def test_no_transformation(self):
        """File without test prefix/suffix returns None."""
        source_files = {"src/utils.py"}
        result = _match_test_to_source_path("src/utils.py", source_files)
        self.assertIsNone(result)


class TestResolveImportsNonRelative(unittest.TestCase):
    """Tests for resolve_imports — non-relative paths, Go modules, inheritance."""

    def test_module_to_file_resolution(self):
        symbols = []
        edges = [Edge("file:main.py", "module:pkg.utils", "imports")]
        file_map = {"main.py": "python", "pkg/utils.py": "python"}
        result = resolve_imports(symbols, edges, file_map, [])
        resolved = [e for e in result if e.type == "imports"]
        self.assertTrue(any(e.target == "file:pkg/utils.py" for e in resolved))

    def test_go_module_prefix_stripping(self):
        symbols = []
        edges = [Edge("file:cmd/main.go", "module:github.com/org/repo/pkg/util", "imports")]
        file_map = {"cmd/main.go": "go", "pkg/util/util.go": "go"}
        boundary = ModuleBoundary(root_dir="", build_file="go.mod",
                                  name="github.com/org/repo", language="go")
        result = resolve_imports(symbols, edges, file_map, [boundary])
        resolved = [e for e in result if e.type == "imports"]
        self.assertTrue(any(e.target == "file:pkg/util/util.go" for e in resolved))

    def test_inheritance_resolution(self):
        parent = Symbol(
            id="class:a.py:Base", type="class", name="Base",
            file="a.py", language="python", line_start=1, line_end=10,
        )
        child = Symbol(
            id="class:b.py:Child", type="class", name="Child",
            file="b.py", language="python", line_start=1, line_end=10, bases=["Base"],
        )
        edges = [Edge("class:b.py:Child", "class:?:Base", "inherits", {"unresolved": True})]
        result = resolve_imports([parent, child], edges, {"a.py": "python", "b.py": "python"}, [])
        inherits = [e for e in result if e.type == "inherits"]
        self.assertEqual(len(inherits), 1)
        self.assertEqual(inherits[0].target, "class:a.py:Base")
        self.assertFalse(inherits[0].metadata.get("unresolved", False))

    def test_inheritance_same_file_disambiguation(self):
        """When multiple classes share a name, prefer the one in the same file."""
        base_a = Symbol(
            id="class:a.py:Base", type="class", name="Base",
            file="a.py", language="python", line_start=1, line_end=10,
        )
        base_c = Symbol(
            id="class:c.py:Base", type="class", name="Base",
            file="c.py", language="python", line_start=1, line_end=10,
        )
        child = Symbol(
            id="class:a.py:Child", type="class", name="Child",
            file="a.py", language="python", line_start=15, line_end=25, bases=["Base"],
        )
        edges = [Edge("class:a.py:Child", "class:?:Base", "inherits", {"unresolved": True})]
        result = resolve_imports([base_a, base_c, child], edges,
                                 {"a.py": "python", "c.py": "python"}, [])
        inherits = [e for e in result if e.type == "inherits"]
        self.assertEqual(inherits[0].target, "class:a.py:Base")

    def test_non_import_edges_passed_through(self):
        edges = [
            Edge("file:a.py", "function:a.py:foo", "contains"),
            Edge("function:a.py:foo", "function:b.py:bar", "calls"),
        ]
        result = resolve_imports([], edges, {"a.py": "python", "b.py": "python"}, [])
        types = {e.type for e in result}
        self.assertIn("contains", types)
        self.assertIn("calls", types)


class TestComputeDirectoryStats(unittest.TestCase):
    """Tests for compute_directory_stats."""

    def test_basic_stats(self):
        files = [
            ("/abs/src/a.py", "src/a.py", "python", False),
            ("/abs/src/b.py", "src/b.py", "python", False),
            ("/abs/tests/test_a.py", "tests/test_a.py", "python", True),
        ]
        symbols = [
            Symbol(id="class:src/a.py:Foo", type="class", name="Foo",
                   file="src/a.py", language="python", line_start=1, line_end=10),
            Symbol(id="function:src/a.py:bar", type="function", name="bar",
                   file="src/a.py", language="python", line_start=15, line_end=20),
        ]
        stats = compute_directory_stats(files, symbols)
        self.assertIn("src", stats)
        self.assertIn("tests", stats)
        self.assertEqual(stats["src"]["files"], 2)
        self.assertEqual(stats["src"]["test_files"], 0)
        self.assertEqual(stats["tests"]["test_files"], 1)
        self.assertEqual(stats["src"]["classes"], 1)
        self.assertEqual(stats["src"]["functions"], 1)
        self.assertIn("python", stats["src"]["languages"])


class TestComputeGraphStats(unittest.TestCase):
    """Tests for compute_graph_stats."""

    def test_counts(self):
        files = [
            ("/a", "a.py", "python", False),
            ("/b", "b.go", "go", False),
            ("/t", "test_a.py", "python", True),
        ]
        symbols = [
            Symbol(id="f:a.py:foo", type="function", name="foo",
                   file="a.py", language="python", line_start=1, line_end=5),
            Symbol(id="c:a.py:Bar", type="class", name="Bar",
                   file="a.py", language="python", line_start=10, line_end=20),
        ]
        edges = [
            Edge("file:a.py", "f:a.py:foo", "contains"),
            Edge("file:b.go", "file:a.py", "imports"),
        ]
        stats = compute_graph_stats(symbols, edges, files)
        self.assertEqual(stats["lang_counts"]["python"], 2)
        self.assertEqual(stats["lang_counts"]["go"], 1)
        self.assertEqual(stats["type_counts"]["function"], 1)
        self.assertEqual(stats["type_counts"]["class"], 1)
        self.assertEqual(stats["edge_type_counts"]["contains"], 1)
        self.assertEqual(stats["edge_type_counts"]["imports"], 1)
        self.assertEqual(stats["test_count"], 1)


class TestGenerateTagsJson(unittest.TestCase):
    """Tests for generate_tags_json output."""

    def test_basic_tags(self):
        symbols = [
            Symbol(id="function:b.py:bar", type="function", name="bar",
                   file="b.py", language="python", line_start=5, line_end=15,
                   params=["x", "y"], signature="(x, y)"),
            Symbol(id="class:a.py:Foo", type="class", name="Foo",
                   file="a.py", language="python", line_start=1, line_end=10,
                   bases=["Base"], docstring="A foo class."),
        ]
        tags = generate_tags_json(symbols)
        # Should be sorted by file then line
        self.assertEqual(tags[0]["file"], "a.py")
        self.assertEqual(tags[1]["file"], "b.py")
        # Check fields present
        foo_tag = tags[0]
        self.assertEqual(foo_tag["name"], "Foo")
        self.assertEqual(foo_tag["bases"], ["Base"])
        self.assertEqual(foo_tag["doc"], "A foo class.")
        bar_tag = tags[1]
        self.assertEqual(bar_tag["params"], ["x", "y"])
        self.assertEqual(bar_tag["signature"], "(x, y)")

    def test_private_flag(self):
        sym = Symbol(id="function:a.py:_helper", type="function", name="_helper",
                     file="a.py", language="python", line_start=1, line_end=3,
                     exported=False)
        tags = generate_tags_json([sym])
        self.assertTrue(tags[0].get("private"))

    def test_test_flag(self):
        sym = Symbol(id="function:test_a.py:test_foo", type="function", name="test_foo",
                     file="test_a.py", language="python", line_start=1, line_end=3,
                     is_test=True)
        tags = generate_tags_json([sym])
        self.assertTrue(tags[0].get("is_test"))

    def test_optional_fields_omitted(self):
        sym = Symbol(id="function:a.py:simple", type="function", name="simple",
                     file="a.py", language="python", line_start=1, line_end=2)
        tags = generate_tags_json([sym])
        self.assertNotIn("scope", tags[0])
        self.assertNotIn("params", tags[0])
        self.assertNotIn("bases", tags[0])
        self.assertNotIn("doc", tags[0])
        self.assertNotIn("signature", tags[0])


class TestDetectTestRelationships(unittest.TestCase):
    """Tests for detect_test_relationships beyond qualified matching."""

    def test_path_mirror_strategy(self):
        files = [
            ("/abs/src/utils.py", "src/utils.py", "python", False),
            ("/abs/tests/test_utils.py", "tests/test_utils.py", "python", True),
        ]
        test_edges = detect_test_relationships([], [], files, {})
        path_edges = [e for e in test_edges if e.metadata.get("strategy") == "path_mirror"]
        self.assertEqual(len(path_edges), 1)
        self.assertEqual(path_edges[0].source, "file:tests/test_utils.py")
        self.assertEqual(path_edges[0].target, "file:src/utils.py")

    def test_no_duplicate_edges(self):
        """Same test-source pair should not produce duplicate edges."""
        source_func = Symbol(
            id="function:src/utils.py:parse", type="function", name="parse",
            file="src/utils.py", language="python", line_start=1, line_end=5,
            is_test=False,
        )
        test_func = Symbol(
            id="function:tests/test_utils.py:test_parse", type="function", name="test_parse",
            file="tests/test_utils.py", language="python", line_start=1, line_end=5,
            is_test=True,
        )
        files = [
            ("/abs/src/utils.py", "src/utils.py", "python", False),
            ("/abs/tests/test_utils.py", "tests/test_utils.py", "python", True),
        ]
        test_edges = detect_test_relationships(
            [source_func, test_func], [], files, {},
        )
        # Count unique (source, target) pairs
        pairs = [(e.source, e.target) for e in test_edges]
        self.assertEqual(len(pairs), len(set(pairs)))


class TestResolveCallsBeforeImportNormalization(unittest.TestCase):
    """Validate that call resolution depends on raw module:/symbol: import
    edges.  If resolve_imports() runs first, those edges are rewritten to
    file: targets and resolve_calls() can no longer recover alias info."""

    def _build_scenario(self):
        """Two files: pkg1/a.py defines foo(), main.py imports and calls it.

        Returns (symbols, raw_edges, call_refs, file_map).
        raw_edges contain the original module:/symbol: targets that
        parse_python would emit.
        """
        sym_foo = Symbol(
            id="function:pkg1/a.py:foo", type="function", name="foo",
            file="pkg1/a.py", language="python", line_start=1, line_end=5,
        )
        sym_main = Symbol(
            id="function:main.py:do_work", type="function", name="do_work",
            file="main.py", language="python", line_start=1, line_end=10,
        )
        symbols = [sym_foo, sym_main]

        # Edges as emitted by parse_python (before resolve_imports)
        raw_edges = [
            Edge("file:main.py", "module:pkg1.a", "imports"),
            Edge("file:main.py", "symbol:pkg1.a.foo", "imports",
                 {"from_import": True}),
            Edge("file:main.py", "function:main.py:do_work", "contains"),
            Edge("file:pkg1/a.py", "function:pkg1/a.py:foo", "contains"),
        ]

        # do_work() calls foo() — unqualified, relying on the from-import
        call_refs = [
            CallRef("function:main.py:do_work", "foo", 5, "main.py"),
        ]

        file_map = {"main.py": "python", "pkg1/a.py": "python"}
        return symbols, raw_edges, call_refs, file_map

    def test_resolve_calls_before_imports_succeeds(self):
        """Correct order: resolve_calls THEN resolve_imports."""
        symbols, raw_edges, call_refs, file_map = self._build_scenario()

        # Step 1: resolve calls while raw module:/symbol: edges exist
        call_edges = resolve_calls(call_refs, symbols, raw_edges, file_map)

        # Step 2: normalize imports
        all_edges = raw_edges + call_edges
        all_edges = resolve_imports(symbols, all_edges, file_map, [])

        # The call from do_work -> foo must be present
        calls = [e for e in all_edges if e.type == "calls"]
        self.assertEqual(len(calls), 1,
                         "do_work() -> foo() call edge must be resolved")
        self.assertEqual(calls[0].source, "function:main.py:do_work")
        self.assertEqual(calls[0].target, "function:pkg1/a.py:foo")

    def test_resolve_calls_after_imports_loses_alias(self):
        """Wrong order: resolve_imports THEN resolve_calls — alias is lost."""
        symbols, raw_edges, call_refs, file_map = self._build_scenario()

        # Step 1: normalize imports first (rewrites module:/symbol: -> file:)
        normalized_edges = resolve_imports(symbols, raw_edges, file_map, [])

        # Verify the raw alias edges are gone
        alias_edges = [
            e for e in normalized_edges
            if e.target.startswith("symbol:") or e.target.startswith("module:")
        ]
        self.assertEqual(len(alias_edges), 0,
                         "After resolve_imports, module:/symbol: targets "
                         "should be rewritten")

        # Step 2: resolve calls — can no longer find alias info
        call_edges = resolve_calls(call_refs, symbols, normalized_edges,
                                   file_map)

        # With the bug, foo() is NOT resolvable via import alias because
        # the symbol:pkg1.a.foo edge was rewritten to file:pkg1/a.py or
        # function:pkg1/a.py:foo.  resolve_calls only checks symbol: and
        # module: targets for building file_imports, so alias lookup fails.
        #
        # The call might still resolve via the last-resort global name
        # match (strategy 4) IF foo is unique.  To make the test robust
        # we add a second foo() so global name match is ambiguous.
        pass

    def test_ambiguous_name_needs_alias(self):
        """Two different foo() definitions — only import alias disambiguates."""
        sym_foo1 = Symbol(
            id="function:pkg1/a.py:foo", type="function", name="foo",
            file="pkg1/a.py", language="python", line_start=1, line_end=5,
        )
        sym_foo2 = Symbol(
            id="function:pkg2/b.py:foo", type="function", name="foo",
            file="pkg2/b.py", language="python", line_start=1, line_end=5,
        )
        sym_main = Symbol(
            id="function:main.py:do_work", type="function", name="do_work",
            file="main.py", language="python", line_start=1, line_end=10,
        )
        symbols = [sym_foo1, sym_foo2, sym_main]

        raw_edges = [
            Edge("file:main.py", "module:pkg1.a", "imports"),
            Edge("file:main.py", "symbol:pkg1.a.foo", "imports",
                 {"from_import": True}),
            Edge("file:main.py", "function:main.py:do_work", "contains"),
        ]

        call_refs = [
            CallRef("function:main.py:do_work", "foo", 5, "main.py"),
        ]
        file_map = {
            "main.py": "python",
            "pkg1/a.py": "python",
            "pkg2/b.py": "python",
        }

        # Correct order: calls first, imports second
        call_edges = resolve_calls(call_refs, symbols, raw_edges, file_map)
        self.assertEqual(len(call_edges), 1,
                         "With raw import edges, alias resolves to pkg1.a.foo")
        self.assertEqual(call_edges[0].target, "function:pkg1/a.py:foo")

        # Wrong order: imports first, calls second
        normalized = resolve_imports(symbols, list(raw_edges), file_map, [])
        bad_call_edges = resolve_calls(call_refs, symbols, normalized,
                                       file_map)
        # With two foo() definitions and no alias info, last-resort name
        # match refuses to guess (candidates > 1), so the call is lost.
        self.assertEqual(len(bad_call_edges), 0,
                         "After import normalization, ambiguous foo() cannot "
                         "be resolved — demonstrates the bug")

    def test_module_import_alias_resolution(self):
        """import pkg1.a; pkg1.a.foo() should resolve via module: edge."""
        sym_foo = Symbol(
            id="function:pkg1/a.py:foo", type="function", name="foo",
            file="pkg1/a.py", language="python", line_start=1, line_end=5,
        )
        sym_main = Symbol(
            id="function:main.py:caller", type="function", name="caller",
            file="main.py", language="python", line_start=1, line_end=10,
        )
        symbols = [sym_foo, sym_main]

        raw_edges = [
            Edge("file:main.py", "module:pkg1.a", "imports"),
        ]
        call_refs = [
            CallRef("function:main.py:caller", "a.foo", 5, "main.py"),
        ]
        file_map = {"main.py": "python", "pkg1/a.py": "python"}

        # Correct order
        call_edges = resolve_calls(call_refs, symbols, raw_edges, file_map)
        self.assertEqual(len(call_edges), 1)
        self.assertEqual(call_edges[0].target, "function:pkg1/a.py:foo")

    def test_end_to_end_parse_and_resolve(self):
        """Full pipeline: parse Python, resolve calls, resolve imports."""
        code_a = textwrap.dedent("""\
            def foo():
                return 42
        """)
        code_main = textwrap.dedent("""\
            from pkg1.a import foo

            def do_work():
                result = foo()
                return result
        """)

        syms_a, edges_a, calls_a = parse_python(code_a, "pkg1/a.py", False)
        syms_main, edges_main, calls_main = parse_python(
            code_main, "main.py", False,
        )

        all_symbols = syms_a + syms_main
        all_edges = edges_a + edges_main
        all_calls = calls_a + calls_main
        file_map = {"main.py": "python", "pkg1/a.py": "python"}

        # Verify parse produced the expected raw import edges
        symbol_imports = [
            e for e in all_edges
            if e.target.startswith("symbol:") and e.type == "imports"
        ]
        self.assertTrue(len(symbol_imports) > 0,
                        "parse_python should emit symbol: import edges")

        # Correct order: calls before imports
        call_edges = resolve_calls(all_calls, all_symbols, all_edges,
                                   file_map)
        all_edges.extend(call_edges)
        all_edges = resolve_imports(all_symbols, all_edges, file_map, [])

        calls = [e for e in all_edges if e.type == "calls"]
        self.assertTrue(
            any(e.source == "function:main.py:do_work"
                and e.target == "function:pkg1/a.py:foo"
                for e in calls),
            "do_work -> foo call edge must survive the full pipeline",
        )


if __name__ == "__main__":
    unittest.main()