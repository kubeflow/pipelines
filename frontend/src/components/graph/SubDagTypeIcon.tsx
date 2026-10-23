// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import { Layers, Repeat, Repeat1, Diamond } from 'lucide-react';
import type { SubDagKind } from './Constants';

export default function SubDagTypeIcon({ kind }: { kind?: SubDagKind }) {
  const Icon =
    kind === 'Loop'
      ? Repeat
      : kind === 'Iteration'
        ? Repeat1
        : kind === 'Condition'
          ? Diamond
          : Layers;
  return (
    <Icon
      size={16}
      role='img'
      aria-label={kind === 'Condition' ? 'Conditional' : kind || 'Sub-DAG'}
      className='kfp-graph-type-icon'
    />
  );
}
