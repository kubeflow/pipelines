#!/usr/bin/env python3
"""query_graph.py - Query the code-tree knowledge graph.

Traverses the knowledge graph to find relevant code and extracts
clean, chunked snippets for AI agent consumption.

Usage:
    python query_graph.py --symbol ClassName          # Find symbol definition
    python query_graph.py --deps path/to/file.py      # File dependencies
    python query_graph.py --rdeps path/to/file.py     # Reverse dependencies
    python query_graph.py --hierarchy ClassName        # Inheritance tree
    python query_graph.py --search "keyword"           # Search symbols
    python query_graph.py --module src/api             # Module overview
    python query_graph.py --entry-points               # Find entry points
    python query_graph.py --chunks path/to/file.py     # Extract code chunks

All queries output file:line references suitable for AI agent context.
"""

import argparse
import json
import os
import sys
from collections import defaultdict
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
    """Query interface for the code-tree knowledge graph."""

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

        for edge in self.graph.get("edges", []):
            src = edge.get("source", "")
            tgt = edge.get("target", "")
            self.outgoing[src].append((tgt, edge))
            self.incoming[tgt].append((src, edge))

    def find_symbol(self, name: str) -> list[dict]:
        """Find all symbols matching a name (exact or partial)."""
        results = []

        # Exact match
        if name in self.nodes_by_name:
            results.extend(self.nodes_by_name[name])

        # Partial match (case-insensitive)
        if not results:
            name_lower = name.lower()
            for n, nodes in self.nodes_by_name.items():
                if name_lower in n.lower():
                    results.extend(nodes)

        return results

    def get_dependencies(self, file_path: str) -> dict:
        """Get all files that a given file depends on."""
        file_id = f"file:{file_path}"
        deps = {"imports": [], "inherits": [], "implements": []}

        for target, edge in self.outgoing.get(file_id, []):
            if edge["type"] == "imports" and target.startswith("file:"):
                deps["imports"].append(target[5:])

        # Also check symbols in this file for inheritance
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

        return deps

    def get_reverse_dependencies(self, file_path: str) -> list[str]:
        """Get all files that import a given file."""
        file_id = f"file:{file_path}"
        rdeps = set()

        for source, edge in self.incoming.get(file_id, []):
            if edge["type"] == "imports" and source.startswith("file:"):
                rdeps.add(source[5:])

        # Also check symbols in this file
        for node in self.nodes_by_file.get(file_path, []):
            nid = node.get("id", "")
            for source, edge in self.incoming.get(nid, []):
                if edge["type"] == "imports" and source.startswith("file:"):
                    rdeps.add(source[5:])

        return sorted(rdeps)

    def get_hierarchy(self, class_name: str) -> dict:
        """Get the inheritance hierarchy for a class."""
        # Find the class node
        candidates = [
            n for n in self.nodes_by_name.get(class_name, [])
            if n.get("type") in ("class", "struct", "interface")
        ]

        if not candidates:
            return {"error": f"Class '{class_name}' not found"}

        result = {"ancestors": [], "descendants": []}

        for cls in candidates:
            cls_id = cls.get("id", "")

            # Find ancestors (what this class inherits from)
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

            # Find descendants (what inherits from this class)
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

    def get_module(self, module_path: str) -> dict:
        """Get overview of a module (directory)."""
        module_data = self.modules.get("modules", {}).get(module_path, {})

        # Find all symbols in this module
        module_symbols = []
        for tag in self.tags:
            if tag.get("file", "").startswith(module_path):
                module_symbols.append(tag)

        return {
            "path": module_path,
            "files": module_data.get("files", []),
            "depends_on": module_data.get("depends_on", []),
            "symbols": module_symbols,
        }

    def find_entry_points(self) -> list[dict]:
        """Find likely entry points (files with no internal importers)."""
        all_files = {
            n.get("path") for n in self.nodes_by_type.get("file", []) if n.get("path")
        }

        imported_files = set()
        for edge in self.graph.get("edges", []):
            if edge["type"] == "imports" and edge["target"].startswith("file:"):
                imported_files.add(edge["target"][5:])

        # Entry points = files that are not imported by others
        entry_candidates = all_files - imported_files

        # Prioritize files with main-like patterns
        main_patterns = ["main.py", "main.go", "index.js", "index.ts", "app.py", "server.go", "__main__.py", "cli.py", "cmd/"]
        results = []

        for f in sorted(entry_candidates):
            priority = 0
            for pat in main_patterns:
                if pat in f:
                    priority = 10
                    break

            file_symbols = [t for t in self.tags if t.get("file") == f]
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

        # Get symbols in this file, sorted by line
        file_symbols = sorted(
            [t for t in self.tags if t.get("file") == file_path],
            key=lambda t: t.get("line", 0),
        )

        if not file_symbols:
            # No symbols extracted; return whole file as one chunk
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
            # Find end: next symbol's start or +50 lines
            line_end = line_start + 50
            for other in file_symbols:
                other_start = other.get("line", 0)
                if other_start > line_start:
                    line_end = other_start - 1
                    break
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

    if node.get("params"):
        parts.append(f"  Params: {', '.join(node['params'])}")
    if node.get("bases"):
        parts.append(f"  Bases: {', '.join(node['bases'])}")
    if node.get("decorators"):
        parts.append(f"  Decorators: {', '.join(node['decorators'])}")
    if node.get("docstring") or node.get("doc"):
        doc = node.get("docstring") or node.get("doc", "")
        parts.append(f"  Doc: {doc[:120]}")

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


# ── Main ──────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Query the code-tree knowledge graph",
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
    group.add_argument("--stats", action="store_true", help="Show graph statistics")

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
                print(f"  {r['type']:10s} {r['name']:30s} {r['file']}:{r['line']}")
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
            print(f"\nSymbols ({len(module_data['symbols'])}):")
            for s in module_data["symbols"][:30]:
                print(f"  {s['type']:10s} {s['name']:30s} :{s['line']}")
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

    elif args.stats:
        meta = gq.graph.get("metadata", {})
        if args.json:
            print(json.dumps(meta, indent=2))
        else:
            print("Knowledge Graph Statistics:\n")
            print(f"  Files:   {meta.get('total_files', '?')}")
            print(f"  Symbols: {meta.get('total_symbols', '?')}")
            print(f"  Languages: {meta.get('languages', {})}")
            print(f"  Symbol types: {meta.get('symbol_types', {})}")
            print(f"  Generated: {meta.get('generated_at', '?')}")


if __name__ == "__main__":
    main()
