#!/usr/bin/env python3
"""Unit tests for query_graph.py — covers start-ID resolution, BFS depth
limiting, edge-type filtering, and output determinism."""

import json
import os
import tempfile
import unittest

from query_graph import GraphQuery


def _build_test_graph():
    """Build a minimal but representative graph for testing."""
    return {
        "metadata": {
            "repo_root": "/fake",
            "total_files": 5,
            "total_symbols": 8,
            "languages": {"python": 3, "go": 2},
            "symbol_types": {"function": 4, "class": 2, "method": 2},
            "edge_types": {"imports": 4, "calls": 3, "tests": 2, "inherits": 1, "contains": 5},
            "generated_at": "2026-01-01T00:00:00",
        },
        "nodes": [
            {"id": "file:src/a.py", "type": "file", "name": "a.py", "path": "src/a.py", "language": "python"},
            {"id": "file:src/b.py", "type": "file", "name": "b.py", "path": "src/b.py", "language": "python"},
            {"id": "file:src/c.py", "type": "file", "name": "c.py", "path": "src/c.py", "language": "python"},
            {"id": "file:tests/test_a.py", "type": "file", "name": "test_a.py", "path": "tests/test_a.py", "language": "python", "is_test": True},
            {"id": "file:pkg/d.go", "type": "file", "name": "d.go", "path": "pkg/d.go", "language": "go"},
            {"id": "function:src/a.py:foo", "type": "function", "name": "foo", "file": "src/a.py", "language": "python", "line_start": 10, "line_end": 20},
            {"id": "function:src/b.py:bar", "type": "function", "name": "bar", "file": "src/b.py", "language": "python", "line_start": 5, "line_end": 15},
            {"id": "function:src/c.py:baz", "type": "function", "name": "baz", "file": "src/c.py", "language": "python", "line_start": 1, "line_end": 8},
            {"id": "class:src/a.py:Base", "type": "class", "name": "Base", "file": "src/a.py", "language": "python", "line_start": 25, "line_end": 50},
            {"id": "class:src/b.py:Child", "type": "class", "name": "Child", "file": "src/b.py", "language": "python", "line_start": 20, "line_end": 40, "bases": ["Base"]},
            {"id": "method:src/b.py:Child.process", "type": "method", "name": "process", "file": "src/b.py", "language": "python", "line_start": 25, "line_end": 35, "scope": "Child"},
            {"id": "function:pkg/d.go:Handle", "type": "function", "name": "Handle", "file": "pkg/d.go", "language": "go", "line_start": 10, "line_end": 30},
        ],
        "edges": [
            # imports: b imports a, c imports b, d imports a
            {"source": "file:src/b.py", "target": "file:src/a.py", "type": "imports"},
            {"source": "file:src/c.py", "target": "file:src/b.py", "type": "imports"},
            {"source": "file:pkg/d.go", "target": "file:src/a.py", "type": "imports"},
            # calls: bar -> foo, baz -> bar, Handle -> foo
            {"source": "function:src/b.py:bar", "target": "function:src/a.py:foo", "type": "calls", "metadata": {"line": 8}},
            {"source": "function:src/c.py:baz", "target": "function:src/b.py:bar", "type": "calls", "metadata": {"line": 3}},
            {"source": "function:pkg/d.go:Handle", "target": "function:src/a.py:foo", "type": "calls", "metadata": {"line": 15}},
            # inherits: Child -> Base
            {"source": "class:src/b.py:Child", "target": "class:src/a.py:Base", "type": "inherits"},
            # tests: test_a tests a.py and foo
            {"source": "file:tests/test_a.py", "target": "file:src/a.py", "type": "tests", "metadata": {"strategy": "path_mirror"}},
            {"source": "file:tests/test_a.py", "target": "function:src/a.py:foo", "type": "tests", "metadata": {"strategy": "name_match"}},
            # contains
            {"source": "file:src/a.py", "target": "function:src/a.py:foo", "type": "contains"},
            {"source": "file:src/a.py", "target": "class:src/a.py:Base", "type": "contains"},
            {"source": "file:src/b.py", "target": "function:src/b.py:bar", "type": "contains"},
            {"source": "file:src/b.py", "target": "class:src/b.py:Child", "type": "contains"},
            {"source": "class:src/b.py:Child", "target": "method:src/b.py:Child.process", "type": "contains"},
        ],
    }


def _build_test_tags():
    """Build tags matching the test graph."""
    return [
        {"name": "foo", "type": "function", "file": "src/a.py", "line": 10, "end_line": 20, "language": "python"},
        {"name": "bar", "type": "function", "file": "src/b.py", "line": 5, "end_line": 15, "language": "python"},
        {"name": "baz", "type": "function", "file": "src/c.py", "line": 1, "end_line": 8, "language": "python"},
        {"name": "Base", "type": "class", "file": "src/a.py", "line": 25, "end_line": 50, "language": "python"},
        {"name": "Child", "type": "class", "file": "src/b.py", "line": 20, "end_line": 40, "language": "python", "bases": ["Base"]},
        {"name": "process", "type": "method", "file": "src/b.py", "line": 25, "end_line": 35, "language": "python", "scope": "Child"},
        {"name": "Handle", "type": "function", "file": "pkg/d.go", "line": 10, "end_line": 30, "language": "go"},
    ]


class TestGraphQuery(unittest.TestCase):
    """Tests for GraphQuery using an in-memory test graph."""

    @classmethod
    def setUpClass(cls):
        """Write test graph artifacts to a temp directory."""
        cls.tmpdir = tempfile.mkdtemp()
        with open(os.path.join(cls.tmpdir, "graph.json"), "w") as f:
            json.dump(_build_test_graph(), f)
        with open(os.path.join(cls.tmpdir, "tags.json"), "w") as f:
            json.dump(_build_test_tags(), f)
        with open(os.path.join(cls.tmpdir, "modules.json"), "w") as f:
            json.dump({"modules": {
                "src": {"files": ["src/a.py", "src/b.py", "src/c.py"], "depends_on": []},
                "pkg": {"files": ["pkg/d.go"], "depends_on": ["src"]},
            }}, f)
        cls.gq = GraphQuery(cls.tmpdir, "/fake")

    # ── Start-ID resolution ──────────────────────────────────────

    def test_impact_resolves_file_path(self):
        """Impact analysis should resolve a file path to its file node and contained symbols."""
        impact = self.gq.get_impact("src/a.py", max_depth=1)
        self.assertNotIn("error", impact)
        # b.py imports a.py, d.go imports a.py
        self.assertIn("src/b.py", impact["affected_files"])
        self.assertIn("pkg/d.go", impact["affected_files"])

    def test_impact_resolves_symbol_name(self):
        """Impact analysis should resolve a symbol name and find its dependents."""
        impact = self.gq.get_impact("foo", max_depth=1)
        self.assertNotIn("error", impact)
        # bar calls foo, Handle calls foo
        affected_names = [s["name"] for s in impact["affected_symbols"]]
        self.assertIn("bar", affected_names)
        self.assertIn("Handle", affected_names)

    def test_impact_unknown_target_returns_error(self):
        """Impact analysis on a nonexistent target should return an error."""
        impact = self.gq.get_impact("nonexistent_symbol_xyz")
        self.assertIn("error", impact)

    # ── BFS depth limiting ───────────────────────────────────────

    def test_impact_depth_1_excludes_transitive(self):
        """Depth 1 should find direct dependents only, not transitive ones."""
        impact = self.gq.get_impact("src/a.py", max_depth=1)
        # c.py depends on b.py which depends on a.py — transitive, should NOT appear at depth 1
        self.assertNotIn("src/c.py", impact["affected_files"])

    def test_impact_depth_2_includes_transitive(self):
        """Depth 2 should find transitive dependents through one hop."""
        impact = self.gq.get_impact("src/a.py", max_depth=2)
        # c.py imports b.py which imports a.py — should appear at depth 2
        self.assertIn("src/c.py", impact["affected_files"])

    def test_impact_depth_0_returns_no_affected(self):
        """Depth 0 should return only origin nodes, no affected items."""
        impact = self.gq.get_impact("src/a.py", max_depth=0)
        self.assertEqual(impact["total_affected_files"], 0)
        self.assertEqual(impact["total_affected_symbols"], 0)

    # ── Edge-type filtering ──────────────────────────────────────

    def test_impact_follows_imports(self):
        """Impact BFS should follow import edges."""
        impact = self.gq.get_impact("src/a.py", max_depth=1)
        self.assertIn("src/b.py", impact["affected_files"])

    def test_impact_follows_calls(self):
        """Impact BFS should follow call edges."""
        impact = self.gq.get_impact("foo", max_depth=1)
        affected_names = [s["name"] for s in impact["affected_symbols"]]
        self.assertIn("bar", affected_names)

    def test_impact_follows_inherits(self):
        """Impact BFS should follow inheritance edges."""
        impact = self.gq.get_impact("Base", max_depth=1)
        affected_names = [s["name"] for s in impact["affected_symbols"]]
        self.assertIn("Child", affected_names)

    def test_impact_does_not_follow_contains(self):
        """Impact BFS should not follow 'contains' edges (not a dependency)."""
        impact = self.gq.get_impact("foo", max_depth=5)
        # 'contains' edge from file:src/a.py -> function:foo should NOT make
        # a.py appear as affected by foo
        affected_names = [s["name"] for s in impact["affected_symbols"]]
        self.assertNotIn("Base", affected_names)  # Base is contained by a.py, not dependent on foo

    def test_impact_does_not_follow_tests(self):
        """Impact BFS should not follow 'tests' edges as dependency edges."""
        # tests edges are separate from dependency edges
        impact = self.gq.get_impact("src/a.py", max_depth=1)
        # test_a.py tests a.py, but tests is not a dependency direction for impact
        affected_names = [s["name"] for s in impact["affected_symbols"]]
        # The test file should NOT appear in affected_symbols (it's not importing/calling a.py)
        # But it appears in affected_files via import — let's check edge types used
        # Actually tests/test_a.py has a "tests" edge, not "imports", so it should NOT appear
        self.assertNotIn("tests/test_a.py", impact["affected_files"])

    # ── Test impact ──────────────────────────────────────────────

    def test_test_impact_finds_test_files(self):
        """Test impact should find tests for affected code."""
        ti = self.gq.get_test_impact("src/a.py")
        self.assertNotIn("error", ti)
        self.assertIn("tests/test_a.py", ti["affected_test_files"])

    def test_test_impact_by_symbol(self):
        """Test impact should work with symbol names."""
        ti = self.gq.get_test_impact("foo")
        self.assertNotIn("error", ti)
        self.assertGreaterEqual(ti["total_test_files"], 1)

    # ── Symbol lookup ────────────────────────────────────────────

    def test_find_symbol_exact(self):
        """Exact symbol lookup should return matching nodes."""
        results = self.gq.find_symbol("foo")
        self.assertTrue(len(results) >= 1)
        self.assertEqual(results[0]["name"], "foo")

    def test_find_symbol_partial(self):
        """Partial symbol lookup should work as fallback."""
        results = self.gq.find_symbol("Handl")
        self.assertTrue(len(results) >= 1)
        names = [r["name"] for r in results]
        self.assertIn("Handle", names)

    def test_find_symbol_not_found(self):
        """Missing symbol should return empty list."""
        results = self.gq.find_symbol("totally_nonexistent_xyz_123")
        self.assertEqual(len(results), 0)

    # ── Dependency queries ───────────────────────────────────────

    def test_get_dependencies(self):
        """get_dependencies should return files imported by a given file."""
        deps = self.gq.get_dependencies("src/b.py")
        self.assertIn("src/a.py", deps["imports"])

    def test_get_reverse_dependencies(self):
        """get_reverse_dependencies should return files that import a given file."""
        rdeps = self.gq.get_reverse_dependencies("src/a.py")
        self.assertIn("src/b.py", rdeps)
        self.assertIn("pkg/d.go", rdeps)

    # ── Call graph ───────────────────────────────────────────────

    def test_get_callers(self):
        """get_callers should find functions that call a given symbol."""
        callers = self.gq.get_callers("foo")
        caller_names = [c["caller"] for c in callers]
        self.assertIn("bar", caller_names)
        self.assertIn("Handle", caller_names)

    def test_get_callees(self):
        """get_callees should find functions called by a given symbol."""
        callees = self.gq.get_callees("bar")
        callee_names = [c["callee"] for c in callees]
        self.assertIn("foo", callee_names)

    def test_call_chain(self):
        """get_call_chain should find paths between symbols."""
        chains = self.gq.get_call_chain("baz", "foo")
        self.assertTrue(len(chains) >= 1)
        # baz -> bar -> foo
        self.assertEqual(len(chains[0]), 3)

    # ── Hierarchy ────────────────────────────────────────────────

    def test_hierarchy_descendants(self):
        """Hierarchy should show classes that inherit from a base."""
        h = self.gq.get_hierarchy("Base")
        descendant_names = [d["name"] for d in h["descendants"]]
        self.assertIn("Child", descendant_names)

    def test_hierarchy_ancestors(self):
        """Hierarchy should show parent classes."""
        h = self.gq.get_hierarchy("Child")
        ancestor_names = [a["name"] for a in h["ancestors"]]
        self.assertIn("Base", ancestor_names)

    # ── Output determinism ───────────────────────────────────────

    def test_reverse_deps_sorted(self):
        """Reverse dependencies should be returned in sorted order."""
        rdeps = self.gq.get_reverse_dependencies("src/a.py")
        self.assertEqual(rdeps, sorted(rdeps))

    def test_impact_files_sorted(self):
        """Impact affected_files should be returned in sorted order."""
        impact = self.gq.get_impact("src/a.py", max_depth=3)
        self.assertEqual(impact["affected_files"], sorted(impact["affected_files"]))

    def test_impact_symbols_sorted_by_depth(self):
        """Impact affected_symbols should be sorted by depth."""
        impact = self.gq.get_impact("src/a.py", max_depth=3)
        depths = [s["depth"] for s in impact["affected_symbols"]]
        self.assertEqual(depths, sorted(depths))

    def test_callers_sorted_by_file_and_line(self):
        """Callers should be sorted by file then line for deterministic output."""
        callers = self.gq.get_callers("foo")
        keys = [(str(c.get("file", "")), c.get("line", 0)) for c in callers]
        self.assertEqual(keys, sorted(keys))

    # ── Test coverage ────────────────────────────────────────────

    def test_get_test_coverage(self):
        """get_test_coverage should find tests for a file."""
        cov = self.gq.get_test_coverage("src/a.py")
        self.assertGreaterEqual(cov["total"], 1)
        test_files = [t["test"] for t in cov["tests"]]
        self.assertTrue(any("test_a" in t for t in test_files))

    # ── Search ───────────────────────────────────────────────────

    def test_search_by_name(self):
        """Search should find symbols by name."""
        results = self.gq.search("bar")
        names = [r["name"] for r in results]
        self.assertIn("bar", names)

    def test_search_by_file_path(self):
        """Search should match on file paths."""
        results = self.gq.search("pkg/d")
        self.assertTrue(len(results) >= 1)

    # ── Stats ────────────────────────────────────────────────────

    def test_graph_metadata_present(self):
        """Graph metadata should be accessible."""
        meta = self.gq.graph.get("metadata", {})
        self.assertEqual(meta["total_files"], 5)
        self.assertEqual(meta["total_symbols"], 8)


if __name__ == "__main__":
    unittest.main()
