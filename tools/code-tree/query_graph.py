#!/usr/bin/env python3
"""query_graph.py - Query the code-tree knowledge graph with impact analysis.

Traverses the knowledge graph to find relevant code, trace call chains,
analyze change impact, and map test coverage.

Usage:
    python query_graph.py --symbol ClassName          # Find symbol definition
    python query_graph.py --deps path/to/file.py      # File dependencies
    python query_graph.py --rdeps path/to/file.py     # Reverse dependencies
    python query_graph.py --hierarchy ClassName        # Inheritance tree
    python query_graph.py --search "keyword"           # Search symbols
    python query_graph.py --module src/api             # Module overview
    python query_graph.py --entry-points               # Find entry points
    python query_graph.py --chunks path/to/file.py     # Extract code chunks
    python query_graph.py --callers func_name          # Who calls this?
    python query_graph.py --callees func_name          # What does this call?
    python query_graph.py --impact path/to/file.py     # Change impact analysis
    python query_graph.py --test-impact path/to/file.py  # Which tests are affected?

All queries output file:line references suitable for AI agent context.
"""

import argparse
import json
import os
import sys
from collections import defaultdict, deque
from pathlib import Path
from typing import Optional


def load_graph(graph_dir: str) -> dict:
    graph_path = os.path.join(graph_dir, "graph.json")
    if not os.path.exists(graph_path):
        print(f"Error: graph.json not found in {graph_dir}", file=sys.stderr)
        print("Run code_tree.py first to generate the knowledge graph.", file=sys.stderr)
        sys.exit(1)
    with open(graph_path, "r") as f:
        return json.load(f)


def load_tags(graph_dir: str) -> list[dict]:
    tags_path = os.path.join(graph_dir, "tags.json")
    if not os.path.exists(tags_path):
        return []
    with open(tags_path, "r") as f:
        return json.load(f)


def load_modules(graph_dir: str) -> dict:
    modules_path = os.path.join(graph_dir, "modules.json")
    if not os.path.exists(modules_path):
        return {"modules": {}}
    with open(modules_path, "r") as f:
        return json.load(f)


class GraphQuery:
    """Query interface for the code-tree knowledge graph with impact analysis."""

    def __init__(self, graph_dir: str, repo_root: str):
        self.graph = load_graph(graph_dir)
        self.tags = load_tags(graph_dir)
        self.modules = load_modules(graph_dir)
        self.repo_root = repo_root

        # Build indexes
        self.nodes_by_id = {}
        self.nodes_by_name = defaultdict(list)
        self.nodes_by_file = defaultdict(list)
        self.nodes_by_type = defaultdict(list)

        for node in self.graph.get("nodes", []):
            nid = node.get("id", "")
            self.nodes_by_id[nid] = node
            name = node.get("name", "")
            self.nodes_by_name[name].append(node)
            file_path = node.get("file") or node.get("path", "")
            if file_path:
                self.nodes_by_file[file_path].append(node)
            self.nodes_by_type[node.get("type", "")].append(node)

        # Build adjacency lists
        self.outgoing = defaultdict(list)  # source -> [(target, edge)]
        self.incoming = defaultdict(list)  # target -> [(source, edge)]

        # Also index by edge type for fast filtering
        self.outgoing_by_type = defaultdict(lambda: defaultdict(list))
        self.incoming_by_type = defaultdict(lambda: defaultdict(list))

        for edge in self.graph.get("edges", []):
            src = edge.get("source", "")
            tgt = edge.get("target", "")
            etype = edge.get("type", "")
            self.outgoing[src].append((tgt, edge))
            self.incoming[tgt].append((src, edge))
            self.outgoing_by_type[etype][src].append((tgt, edge))
            self.incoming_by_type[etype][tgt].append((src, edge))

    # ── Symbol Lookup ─────────────────────────────────────────────

    def find_symbol(self, name: str) -> list[dict]:
        """Find all symbols matching a name (exact or partial)."""
        results = []

        if name in self.nodes_by_name:
            results.extend(self.nodes_by_name[name])

        if not results:
            name_lower = name.lower()
            for n, nodes in self.nodes_by_name.items():
                if name_lower in n.lower():
                    results.extend(nodes)

        return results

    # ── Dependency Queries ────────────────────────────────────────

    def get_dependencies(self, file_path: str) -> dict:
        """Get all files/symbols that a given file depends on."""
        file_id = f"file:{file_path}"
        deps = {"imports": [], "inherits": [], "calls": []}

        for target, edge in self.outgoing.get(file_id, []):
            if edge["type"] == "imports" and target.startswith("file:"):
                deps["imports"].append(target[5:])

        # Check symbols in this file
        for node in self.nodes_by_file.get(file_path, []):
            nid = node.get("id", "")
            for target, edge in self.outgoing.get(nid, []):
                if edge["type"] == "inherits":
                    target_node = self.nodes_by_id.get(target, {})
                    if target_node:
                        deps["inherits"].append({
                            "from": node.get("name"),
                            "to": target_node.get("name"),
                            "file": target_node.get("file"),
                        })
                elif edge["type"] == "calls":
                    target_node = self.nodes_by_id.get(target, {})
                    if target_node and target_node.get("file") != file_path:
                        deps["calls"].append({
                            "caller": node.get("name"),
                            "callee": target_node.get("name"),
                            "file": target_node.get("file"),
                        })

        return deps

    def get_reverse_dependencies(self, file_path: str) -> list[str]:
        """Get all files that import a given file."""
        file_id = f"file:{file_path}"
        rdeps = set()

        for source, edge in self.incoming.get(file_id, []):
            if edge["type"] == "imports" and source.startswith("file:"):
                rdeps.add(source[5:])

        for node in self.nodes_by_file.get(file_path, []):
            nid = node.get("id", "")
            for source, edge in self.incoming.get(nid, []):
                if edge["type"] == "imports" and source.startswith("file:"):
                    rdeps.add(source[5:])

        return sorted(rdeps)

    # ── Call Graph Queries ────────────────────────────────────────

    def get_callers(self, symbol_name: str) -> list[dict]:
        """Find all functions/methods that call a given symbol."""
        # Find the target symbol(s)
        targets = [
            n for n in self.nodes_by_name.get(symbol_name, [])
            if n.get("type") in ("function", "method", "class")
        ]

        results = []
        seen = set()

        for target in targets:
            tid = target.get("id", "")
            for source_id, edge in self.incoming_by_type["calls"].get(tid, []):
                if source_id in seen:
                    continue
                seen.add(source_id)
                source_node = self.nodes_by_id.get(source_id, {})
                results.append({
                    "caller": source_node.get("name", "?"),
                    "type": source_node.get("type", "?"),
                    "file": source_node.get("file", "?"),
                    "line": source_node.get("line_start", "?"),
                    "call_line": edge.get("metadata", {}).get("line", "?"),
                })

        return sorted(results, key=lambda x: (str(x.get("file", "")), x.get("line", 0)))

    def get_callees(self, symbol_name: str) -> list[dict]:
        """Find all functions/methods called by a given symbol."""
        sources = [
            n for n in self.nodes_by_name.get(symbol_name, [])
            if n.get("type") in ("function", "method")
        ]

        results = []
        seen = set()

        for source in sources:
            sid = source.get("id", "")
            for target_id, edge in self.outgoing_by_type["calls"].get(sid, []):
                if target_id in seen:
                    continue
                seen.add(target_id)
                target_node = self.nodes_by_id.get(target_id, {})
                results.append({
                    "callee": target_node.get("name", "?"),
                    "type": target_node.get("type", "?"),
                    "file": target_node.get("file", "?"),
                    "line": target_node.get("line_start", "?"),
                    "call_line": edge.get("metadata", {}).get("line", "?"),
                })

        return sorted(results, key=lambda x: (str(x.get("file", "")), x.get("line", 0)))

    def get_call_chain(self, from_name: str, to_name: str, max_depth: int = 8) -> list[list[str]]:
        """Find call paths from one symbol to another (BFS shortest paths)."""
        # Find source and target IDs
        from_ids = {
            n["id"] for n in self.nodes_by_name.get(from_name, [])
            if n.get("type") in ("function", "method")
        }
        to_ids = {
            n["id"] for n in self.nodes_by_name.get(to_name, [])
            if n.get("type") in ("function", "method", "class")
        }

        if not from_ids or not to_ids:
            return []

        # BFS from each source
        paths = []
        for start in from_ids:
            queue = deque([(start, [start])])
            visited = {start}

            while queue and len(paths) < 5:
                current, path = queue.popleft()
                if len(path) > max_depth:
                    continue

                for target_id, edge in self.outgoing_by_type["calls"].get(current, []):
                    if target_id in to_ids:
                        paths.append(path + [target_id])
                        continue
                    if target_id not in visited:
                        visited.add(target_id)
                        queue.append((target_id, path + [target_id]))

        return paths

    # ── Hierarchy ─────────────────────────────────────────────────

    def get_hierarchy(self, class_name: str) -> dict:
        """Get the inheritance hierarchy for a class."""
        candidates = [
            n for n in self.nodes_by_name.get(class_name, [])
            if n.get("type") in ("class", "struct", "interface")
        ]

        if not candidates:
            return {"error": f"Class '{class_name}' not found"}

        result = {"ancestors": [], "descendants": []}

        for cls in candidates:
            cls_id = cls.get("id", "")
            visited = set()

            def find_ancestors(node_id, depth=0):
                if node_id in visited or depth > 10:
                    return
                visited.add(node_id)
                for target, edge in self.outgoing.get(node_id, []):
                    if edge["type"] == "inherits":
                        target_node = self.nodes_by_id.get(target, {"name": target.split(":")[-1]})
                        result["ancestors"].append({
                            "name": target_node.get("name", "?"),
                            "file": target_node.get("file", "?"),
                            "depth": depth + 1,
                        })
                        find_ancestors(target, depth + 1)

            find_ancestors(cls_id)
            visited.clear()

            def find_descendants(node_id, depth=0):
                if node_id in visited or depth > 10:
                    return
                visited.add(node_id)
                for source, edge in self.incoming.get(node_id, []):
                    if edge["type"] == "inherits":
                        source_node = self.nodes_by_id.get(source, {"name": source.split(":")[-1]})
                        result["descendants"].append({
                            "name": source_node.get("name", "?"),
                            "file": source_node.get("file", "?"),
                            "depth": depth + 1,
                        })
                        find_descendants(source, depth + 1)

            find_descendants(cls_id)

        return result

    # ── Search ────────────────────────────────────────────────────

    def search(self, keyword: str) -> list[dict]:
        """Search symbols by keyword (name, docstring, file path)."""
        keyword_lower = keyword.lower()
        results = []

        for tag in self.tags:
            score = 0
            name = tag.get("name", "")
            doc = tag.get("doc", "")
            file_path = tag.get("file", "")

            if keyword_lower == name.lower():
                score = 100
            elif keyword_lower in name.lower():
                score = 50
            elif keyword_lower in file_path.lower():
                score = 20
            elif doc and keyword_lower in doc.lower():
                score = 10

            if score > 0:
                results.append({**tag, "_score": score})

        results.sort(key=lambda x: (-x["_score"], x.get("file", ""), x.get("line", 0)))
        return results[:30]

    # ── Module Queries ────────────────────────────────────────────

    def get_module(self, module_path: str) -> dict:
        """Get overview of a module (directory)."""
        module_data = self.modules.get("modules", {}).get(module_path, {})

        module_symbols = []
        for tag in self.tags:
            if tag.get("file", "").startswith(module_path):
                module_symbols.append(tag)

        return {
            "path": module_path,
            "files": module_data.get("files", []),
            "depends_on": module_data.get("depends_on", []),
            "tested_by": module_data.get("tested_by", []),
            "symbols": module_symbols,
        }

    def find_entry_points(self) -> list[dict]:
        """Find likely entry points (files with no internal importers)."""
        all_files = {
            n.get("path") for n in self.nodes_by_type.get("file", []) if n.get("path")
        }

        # Use the imports index instead of scanning all edges
        imported_files = set()
        for file_id, edges_list in self.incoming_by_type["imports"].items():
            if file_id.startswith("file:"):
                imported_files.add(file_id[5:])

        entry_candidates = all_files - imported_files

        main_patterns = [
            "main.py", "main.go", "index.js", "index.ts", "app.py",
            "server.go", "__main__.py", "cli.py", "cmd/",
        ]

        # Build a file-to-tags index for efficient lookup
        tags_by_file = defaultdict(list)
        for t in self.tags:
            f = t.get("file")
            if f:
                tags_by_file[f].append(t)

        results = []

        for f in sorted(entry_candidates):
            priority = 0
            for pat in main_patterns:
                if pat in f:
                    priority = 10
                    break

            file_symbols = tags_by_file.get(f, [])
            has_main = any(t.get("name") == "main" for t in file_symbols)
            if has_main:
                priority = 20

            results.append({
                "file": f,
                "priority": priority,
                "symbols": len(file_symbols),
            })

        results.sort(key=lambda x: (-x["priority"], x["file"]))
        return results[:20]

    # ── Chunk Extraction ──────────────────────────────────────────

    def extract_chunks(self, file_path: str) -> list[dict]:
        """Extract clean code chunks from a file using symbol boundaries."""
        abs_path = os.path.join(self.repo_root, file_path)
        if not os.path.exists(abs_path):
            return [{"error": f"File not found: {file_path}"}]

        try:
            with open(abs_path, "r", encoding="utf-8", errors="ignore") as f:
                lines = f.readlines()
        except OSError:
            return [{"error": f"Cannot read: {file_path}"}]

        file_symbols = sorted(
            [t for t in self.tags if t.get("file") == file_path],
            key=lambda t: t.get("line", 0),
        )

        if not file_symbols:
            return [{
                "file": file_path,
                "line_start": 1,
                "line_end": len(lines),
                "type": "file",
                "name": Path(file_path).name,
                "content": "".join(lines[:200]),
            }]

        chunks = []
        for sym in file_symbols:
            line_start = sym.get("line", 1)
            line_end = sym.get("end_line", line_start + 50)

            # Use actual end_line from tags if available
            line_end = min(line_end, len(lines))

            chunk_lines = lines[line_start - 1 : line_end]
            chunks.append({
                "file": file_path,
                "line_start": line_start,
                "line_end": line_end,
                "type": sym.get("type", "unknown"),
                "name": sym.get("name", "?"),
                "scope": sym.get("scope"),
                "content": "".join(chunk_lines),
            })

        return chunks

    # ── Impact Analysis ───────────────────────────────────────────

    def _compute_impact_bfs(self, target: str, max_depth: int = 5) -> tuple[dict, set]:
        """Core BFS for impact analysis. Returns (affected_map, start_ids).

        affected_map: node_id -> {"depth": N, "via": edge_type}
        start_ids: set of origin node IDs
        """
        start_ids = set()

        # Try as file path
        file_id = f"file:{target}"
        if file_id in self.nodes_by_id:
            start_ids.add(file_id)
            for node in self.nodes_by_file.get(target, []):
                start_ids.add(node.get("id", ""))
        else:
            # Try as symbol name
            for node in self.nodes_by_name.get(target, []):
                start_ids.add(node.get("id", ""))
                if node.get("file"):
                    start_ids.add(f"file:{node['file']}")

        if not start_ids:
            return {}, start_ids

        # BFS through reverse dependency edges (imports, calls, inherits)
        affected = {}
        queue = deque()

        for sid in start_ids:
            affected[sid] = {"depth": 0, "via": "origin"}
            queue.append((sid, 0))

        while queue:
            current_id, depth = queue.popleft()
            if depth >= max_depth:
                continue

            for source_id, edge in self.incoming.get(current_id, []):
                etype = edge.get("type", "")
                if etype not in ("imports", "calls", "inherits"):
                    continue

                if source_id not in affected or affected[source_id]["depth"] > depth + 1:
                    affected[source_id] = {"depth": depth + 1, "via": etype}
                    queue.append((source_id, depth + 1))

        return affected, start_ids

    def get_impact(self, target: str, max_depth: int = 5) -> dict:
        """Compute transitive impact of changing a file or symbol.

        Given a file path or symbol name, finds everything that depends on it
        transitively through imports, calls, and inheritance edges.

        Returns a structured impact report with affected items ranked by distance.
        """
        affected, start_ids = self._compute_impact_bfs(target, max_depth)

        if not start_ids:
            return {"error": f"No file or symbol found matching '{target}'"}

        # Organize results
        affected_files = set()
        affected_symbols = []

        for node_id, info in affected.items():
            if info["depth"] == 0:
                continue  # Skip the origin nodes

            node = self.nodes_by_id.get(node_id, {})
            if node.get("type") == "file":
                affected_files.add(node.get("path", ""))
            elif node.get("type") in ("function", "method", "class", "struct", "interface"):
                affected_symbols.append({
                    "name": node.get("name", "?"),
                    "type": node.get("type", "?"),
                    "file": node.get("file", "?"),
                    "line": node.get("line_start", "?"),
                    "depth": info["depth"],
                    "via": info["via"],
                })

                # Track the file too
                if node.get("file"):
                    affected_files.add(node["file"])

        # Sort by depth then name
        affected_symbols.sort(key=lambda x: (x["depth"], str(x.get("file", "")), x.get("name", "")))

        return {
            "target": target,
            "affected_files": sorted(affected_files),
            "affected_symbols": affected_symbols,
            "total_affected_files": len(affected_files),
            "total_affected_symbols": len(affected_symbols),
            "max_depth_reached": max_depth,
        }

    def get_test_impact(self, target: str) -> dict:
        """Find all tests affected by a change to a file or symbol.

        Combines impact analysis with test mapping to determine which tests
        need to be re-run or reviewed when code changes.
        """
        # Reuse the BFS directly — affected already contains all node IDs
        affected, start_ids = self._compute_impact_bfs(target)

        if not start_ids:
            return {"error": f"No file or symbol found matching '{target}'"}

        # Build the impact report for the summary
        impact = self.get_impact(target)

        # All affected node IDs are already in the affected map
        affected_ids = set(affected.keys())

        # Use the tests index for O(1) lookup per affected node
        affected_tests = []
        seen_tests = set()

        for node_id in affected_ids:
            for test_source, edge in self.incoming_by_type["tests"].get(node_id, []):
                if test_source in seen_tests:
                    continue
                seen_tests.add(test_source)
                test_node = self.nodes_by_id.get(test_source, {})

                test_file = test_node.get("file") or test_node.get("path", "")
                if test_source.startswith("file:"):
                    test_file = test_source[5:]

                affected_tests.append({
                    "test": test_file or test_source,
                    "test_name": test_node.get("name", "?"),
                    "test_type": test_node.get("type", "file"),
                    "tests_target": node_id.split(":")[-1] if ":" in node_id else node_id,
                    "strategy": edge.get("metadata", {}).get("strategy", "?"),
                })

        # Deduplicate by test file
        test_files = sorted(set(t["test"] for t in affected_tests))

        return {
            "target": target,
            "affected_test_files": test_files,
            "affected_tests": affected_tests,
            "total_test_files": len(test_files),
            "total_tests": len(affected_tests),
            "impact_summary": {
                "directly_affected_files": len(impact.get("affected_files", [])),
                "directly_affected_symbols": len(impact.get("affected_symbols", [])),
            },
        }

    def get_test_coverage(self, file_path: str) -> dict:
        """Show which tests cover a given file."""
        file_id = f"file:{file_path}"
        tests = []
        seen = set()

        # Direct test edges to this file
        for source_id, edge in self.incoming_by_type["tests"].get(file_id, []):
            if source_id not in seen:
                seen.add(source_id)
                node = self.nodes_by_id.get(source_id, {})
                tests.append({
                    "test": node.get("path") or node.get("file", source_id),
                    "name": node.get("name", "?"),
                    "type": node.get("type", "?"),
                    "strategy": edge.get("metadata", {}).get("strategy", "?"),
                })

        # Tests targeting symbols in this file
        for node in self.nodes_by_file.get(file_path, []):
            nid = node.get("id", "")
            for source_id, edge in self.incoming_by_type["tests"].get(nid, []):
                if source_id not in seen:
                    seen.add(source_id)
                    source_node = self.nodes_by_id.get(source_id, {})
                    tests.append({
                        "test": source_node.get("file", source_id),
                        "name": source_node.get("name", "?"),
                        "type": source_node.get("type", "?"),
                        "targets": node.get("name", "?"),
                        "strategy": edge.get("metadata", {}).get("strategy", "?"),
                    })

        return {
            "file": file_path,
            "tests": tests,
            "total": len(tests),
        }


# ── Output Formatters ─────────────────────────────────────────────

def format_symbol_result(node: dict) -> str:
    parts = []
    sym_type = node.get("type", "?")
    name = node.get("name", "?")
    file_path = node.get("file") or node.get("path", "?")
    line = node.get("line_start") or node.get("line", "?")

    header = f"{sym_type}: {name}"
    if node.get("scope"):
        header = f"{sym_type}: {node['scope']}.{name}"
    parts.append(header)
    parts.append(f"  Location: {file_path}:{line}")

    if node.get("signature"):
        parts.append(f"  Signature: {name}{node['signature']}")
    elif node.get("params"):
        parts.append(f"  Params: {', '.join(node['params'])}")
    if node.get("bases"):
        parts.append(f"  Bases: {', '.join(node['bases'])}")
    if node.get("decorators"):
        parts.append(f"  Decorators: {', '.join(node['decorators'])}")
    if node.get("docstring") or node.get("doc"):
        doc = node.get("docstring") or node.get("doc", "")
        parts.append(f"  Doc: {doc[:120]}")
    if node.get("is_test"):
        parts.append(f"  [TEST]")

    return "\n".join(parts)


def format_deps(deps: dict) -> str:
    lines = []
    if deps.get("imports"):
        lines.append("Imports:")
        for imp in sorted(deps["imports"]):
            lines.append(f"  -> {imp}")
    if deps.get("inherits"):
        lines.append("Inherits:")
        for inh in deps["inherits"]:
            lines.append(f"  {inh['from']} extends {inh['to']} ({inh.get('file', '?')})")
    if deps.get("calls"):
        lines.append("Calls (cross-file):")
        for call in deps["calls"][:20]:
            lines.append(f"  {call['caller']} -> {call['callee']} ({call.get('file', '?')})")
        if len(deps.get("calls", [])) > 20:
            lines.append(f"  ... and {len(deps['calls']) - 20} more")
    if not lines:
        lines.append("No internal dependencies found.")
    return "\n".join(lines)


def format_hierarchy(hierarchy: dict) -> str:
    if "error" in hierarchy:
        return hierarchy["error"]

    lines = []
    if hierarchy.get("ancestors"):
        lines.append("Ancestors (inherits from):")
        for a in hierarchy["ancestors"]:
            indent = "  " * a["depth"]
            lines.append(f"{indent}<- {a['name']} ({a['file']})")
    if hierarchy.get("descendants"):
        lines.append("Descendants (inherited by):")
        for d in hierarchy["descendants"]:
            indent = "  " * d["depth"]
            lines.append(f"{indent}-> {d['name']} ({d['file']})")
    if not lines:
        lines.append("No inheritance relationships found.")
    return "\n".join(lines)


def format_chunks(chunks: list[dict]) -> str:
    lines = []
    for chunk in chunks:
        if "error" in chunk:
            lines.append(chunk["error"])
            continue
        header = f"--- {chunk['type']}: {chunk['name']} ({chunk['file']}:{chunk['line_start']}-{chunk['line_end']}) ---"
        lines.append(header)
        lines.append(chunk["content"])
        lines.append("")
    return "\n".join(lines)


def format_impact(impact: dict) -> str:
    if "error" in impact:
        return impact["error"]

    lines = []
    lines.append(f"Impact analysis for: {impact['target']}")
    lines.append(f"  Affected files: {impact['total_affected_files']}")
    lines.append(f"  Affected symbols: {impact['total_affected_symbols']}")
    lines.append("")

    if impact.get("affected_files"):
        lines.append("Affected files:")
        for f in impact["affected_files"]:
            lines.append(f"  * {f}")
        lines.append("")

    if impact.get("affected_symbols"):
        # Group by depth
        by_depth = defaultdict(list)
        for sym in impact["affected_symbols"]:
            by_depth[sym["depth"]].append(sym)

        for depth in sorted(by_depth.keys()):
            label = "Direct" if depth == 1 else f"Depth {depth}"
            lines.append(f"{label} dependencies:")
            for sym in by_depth[depth]:
                lines.append(f"  {sym['type']:10s} {sym['name']:30s} {sym['file']}:{sym['line']}  (via {sym['via']})")
            lines.append("")

    return "\n".join(lines)


def format_test_impact(test_impact: dict) -> str:
    if "error" in test_impact:
        return test_impact["error"]

    lines = []
    lines.append(f"Test impact for: {test_impact['target']}")
    lines.append(f"  Test files to re-run: {test_impact['total_test_files']}")
    lines.append(f"  Total test mappings: {test_impact['total_tests']}")
    lines.append(f"  Impact scope: {test_impact['impact_summary']['directly_affected_files']} files, "
                 f"{test_impact['impact_summary']['directly_affected_symbols']} symbols")
    lines.append("")

    if test_impact.get("affected_test_files"):
        lines.append("Test files to re-run:")
        for f in test_impact["affected_test_files"]:
            lines.append(f"  * {f}")
        lines.append("")

    if test_impact.get("affected_tests"):
        lines.append("Test details:")
        for t in test_impact["affected_tests"]:
            lines.append(f"  {t['test_name']:30s} tests {t['tests_target']:20s} ({t['strategy']})")
        lines.append("")

    return "\n".join(lines)


# ── Main ──────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Query the code-tree knowledge graph with impact analysis",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "--graph-dir", default=None,
        help="Directory containing graph.json (default: <repo-root>/docs/code-tree)",
    )
    parser.add_argument(
        "--repo-root", default=".",
        help="Repository root directory (default: current directory)",
    )
    parser.add_argument("--json", action="store_true", help="Output as JSON")

    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--symbol", "-s", help="Find a symbol by name")
    group.add_argument("--deps", "-d", help="Show dependencies of a file")
    group.add_argument("--rdeps", "-r", help="Show reverse dependencies of a file")
    group.add_argument("--hierarchy", "-H", help="Show inheritance hierarchy for a class")
    group.add_argument("--search", "-S", help="Search symbols by keyword")
    group.add_argument("--module", "-m", help="Show module (directory) overview")
    group.add_argument("--entry-points", "-e", action="store_true", help="Find entry points")
    group.add_argument("--chunks", "-c", help="Extract code chunks from a file")
    group.add_argument("--callers", help="Find all callers of a symbol")
    group.add_argument("--callees", help="Find all functions called by a symbol")
    group.add_argument("--call-chain", nargs=2, metavar=("FROM", "TO"),
                       help="Find call paths between two symbols")
    group.add_argument("--impact", "-I", help="Compute change impact for a file or symbol")
    group.add_argument("--test-impact", "-T", help="Find tests affected by changes to a file or symbol")
    group.add_argument("--test-coverage", help="Show test coverage for a file")
    group.add_argument("--stats", action="store_true", help="Show graph statistics")

    parser.add_argument("--depth", type=int, default=5,
                       help="Max depth for impact/call-chain analysis (default: 5)")

    args = parser.parse_args()
    repo_root = os.path.abspath(args.repo_root)
    graph_dir = args.graph_dir or os.path.join(repo_root, "docs", "code-tree")

    gq = GraphQuery(graph_dir, repo_root)

    if args.symbol:
        results = gq.find_symbol(args.symbol)
        if args.json:
            print(json.dumps(results, indent=2, default=str))
        elif results:
            for r in results:
                print(format_symbol_result(r))
                print()
        else:
            print(f"No symbols matching '{args.symbol}' found.")

    elif args.deps:
        deps = gq.get_dependencies(args.deps)
        if args.json:
            print(json.dumps(deps, indent=2))
        else:
            print(f"Dependencies of {args.deps}:\n")
            print(format_deps(deps))

    elif args.rdeps:
        rdeps = gq.get_reverse_dependencies(args.rdeps)
        if args.json:
            print(json.dumps(rdeps, indent=2))
        else:
            print(f"Files that import {args.rdeps}:\n")
            for r in rdeps:
                print(f"  <- {r}")
            if not rdeps:
                print("  No internal importers found.")

    elif args.hierarchy:
        hierarchy = gq.get_hierarchy(args.hierarchy)
        if args.json:
            print(json.dumps(hierarchy, indent=2))
        else:
            print(f"Inheritance hierarchy for {args.hierarchy}:\n")
            print(format_hierarchy(hierarchy))

    elif args.search:
        results = gq.search(args.search)
        if args.json:
            print(json.dumps(results, indent=2))
        else:
            print(f"Search results for '{args.search}':\n")
            for r in results:
                test_marker = " [T]" if r.get("is_test") else ""
                print(f"  {r['type']:10s} {r['name']:30s} {r['file']}:{r['line']}{test_marker}")
            if not results:
                print("  No matches found.")

    elif args.module:
        module_data = gq.get_module(args.module)
        if args.json:
            print(json.dumps(module_data, indent=2))
        else:
            print(f"Module: {args.module}\n")
            print(f"Files ({len(module_data['files'])}):")
            for f in module_data["files"]:
                print(f"  {f}")
            print(f"\nDepends on ({len(module_data['depends_on'])}):")
            for d in module_data["depends_on"]:
                print(f"  -> {d}")
            if module_data.get("tested_by"):
                print(f"\nTested by ({len(module_data['tested_by'])}):")
                for t in module_data["tested_by"]:
                    print(f"  <- {t}")
            print(f"\nSymbols ({len(module_data['symbols'])}):")
            for s in module_data["symbols"][:30]:
                test_marker = " [T]" if s.get("is_test") else ""
                print(f"  {s['type']:10s} {s['name']:30s} :{s['line']}{test_marker}")
            if len(module_data["symbols"]) > 30:
                print(f"  ... and {len(module_data['symbols']) - 30} more")

    elif args.entry_points:
        entries = gq.find_entry_points()
        if args.json:
            print(json.dumps(entries, indent=2))
        else:
            print("Entry points (files not imported by others):\n")
            for e in entries:
                marker = " *" if e["priority"] >= 10 else ""
                print(f"  {e['file']}{marker} ({e['symbols']} symbols)")

    elif args.chunks:
        chunks = gq.extract_chunks(args.chunks)
        if args.json:
            print(json.dumps(chunks, indent=2))
        else:
            print(format_chunks(chunks))

    elif args.callers:
        callers = gq.get_callers(args.callers)
        if args.json:
            print(json.dumps(callers, indent=2))
        else:
            print(f"Callers of '{args.callers}':\n")
            if callers:
                for c in callers:
                    print(f"  {c['type']:10s} {c['caller']:30s} {c['file']}:{c['line']}  (calls at :{c['call_line']})")
            else:
                print("  No callers found.")

    elif args.callees:
        callees = gq.get_callees(args.callees)
        if args.json:
            print(json.dumps(callees, indent=2))
        else:
            print(f"Functions called by '{args.callees}':\n")
            if callees:
                for c in callees:
                    print(f"  -> {c['type']:10s} {c['callee']:30s} {c['file']}:{c['line']}")
            else:
                print("  No callees found.")

    elif args.call_chain:
        from_name, to_name = args.call_chain
        chains = gq.get_call_chain(from_name, to_name, args.depth)
        if args.json:
            # Convert node IDs to readable names
            readable_chains = []
            for chain in chains:
                readable = []
                for node_id in chain:
                    node = gq.nodes_by_id.get(node_id, {})
                    readable.append({
                        "name": node.get("name", node_id),
                        "type": node.get("type", "?"),
                        "file": node.get("file", "?"),
                    })
                readable_chains.append(readable)
            print(json.dumps(readable_chains, indent=2))
        else:
            print(f"Call chains from '{from_name}' to '{to_name}':\n")
            if chains:
                for i, chain in enumerate(chains):
                    names = []
                    for node_id in chain:
                        node = gq.nodes_by_id.get(node_id, {})
                        names.append(node.get("name", node_id.split(":")[-1]))
                    print(f"  Chain {i + 1}: {' -> '.join(names)}")
            else:
                print("  No call chains found.")

    elif args.impact:
        impact = gq.get_impact(args.impact, args.depth)
        if args.json:
            print(json.dumps(impact, indent=2))
        else:
            print(format_impact(impact))

    elif args.test_impact:
        test_impact = gq.get_test_impact(args.test_impact)
        if args.json:
            print(json.dumps(test_impact, indent=2))
        else:
            print(format_test_impact(test_impact))

    elif args.test_coverage:
        coverage = gq.get_test_coverage(args.test_coverage)
        if args.json:
            print(json.dumps(coverage, indent=2))
        else:
            print(f"Test coverage for {args.test_coverage}:\n")
            if coverage["tests"]:
                for t in coverage["tests"]:
                    target_str = f" -> {t['targets']}" if t.get("targets") else ""
                    print(f"  {t['name']:30s} ({t['type']}) in {t['test']}{target_str}")
            else:
                print("  No tests found for this file.")

    elif args.stats:
        meta = gq.graph.get("metadata", {})
        if args.json:
            # Don't include file_hashes in stats output (too verbose)
            stats_meta = {k: v for k, v in meta.items() if k != "file_hashes"}
            print(json.dumps(stats_meta, indent=2))
        else:
            print("Knowledge Graph Statistics:\n")
            print(f"  Files:        {meta.get('total_files', '?')}")
            print(f"  Symbols:      {meta.get('total_symbols', '?')}")
            print(f"  Languages:    {meta.get('languages', {})}")
            print(f"  Symbol types: {meta.get('symbol_types', {})}")
            print(f"  Edge types:   {meta.get('edge_types', {})}")
            print(f"  Generated:    {meta.get('generated_at', '?')}")
            if meta.get("module_boundaries"):
                print(f"\n  Build modules:")
                for mb in meta["module_boundaries"]:
                    name = mb.get("name", mb.get("build_file", "?"))
                    print(f"    {name} ({mb.get('build_file', '?')}) in {mb.get('root_dir', '.')}")


if __name__ == "__main__":
    main()
