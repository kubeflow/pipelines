// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

export interface GraphExpansionState {
  collapsed: Set<string>;
  expanded: Set<string>;
  mode: 'default' | 'all' | 'none';
}

const storageKey = (graphId: string) => `kfp:graph-expansion:v1:${graphId}`;
const isStringArray = (value: unknown): value is string[] =>
  Array.isArray(value) && value.every((item) => typeof item === 'string');

export function loadGraphExpansion(graphId?: string): GraphExpansionState {
  try {
    const saved = graphId && JSON.parse(localStorage.getItem(storageKey(graphId)) || 'null');
    if (
      saved &&
      ['default', 'all', 'none'].includes(saved.mode) &&
      isStringArray(saved.collapsed) &&
      isStringArray(saved.expanded)
    ) {
      return {
        mode: saved.mode,
        collapsed: new Set(saved.collapsed),
        expanded: new Set(saved.expanded),
      };
    }
  } catch {
    // Unavailable storage or invalid preferences must not prevent graph rendering.
  }
  return { mode: 'default', collapsed: new Set(), expanded: new Set() };
}

export function saveGraphExpansion(graphId: string | undefined, state: GraphExpansionState): void {
  if (!graphId) return;
  try {
    localStorage.setItem(
      storageKey(graphId),
      JSON.stringify({ ...state, collapsed: [...state.collapsed], expanded: [...state.expanded] }),
    );
  } catch {
    // Retain in-memory interaction when storage is blocked or full.
  }
}
