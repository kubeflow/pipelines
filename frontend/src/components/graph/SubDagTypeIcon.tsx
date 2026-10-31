// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import LayersIcon from '@mui/icons-material/Layers';
import RepeatIcon from '@mui/icons-material/Repeat';
import RepeatOneIcon from '@mui/icons-material/RepeatOne';
import SvgIcon from '@mui/material/SvgIcon';
import type { SubDagKind } from './Constants';

export default function SubDagTypeIcon({ kind }: { kind?: SubDagKind }) {
  const className = 'text-mui-grey-600';
  switch (kind) {
    case 'Loop':
      return <RepeatIcon className={className} titleAccess='Loop' />;
    case 'Iteration':
      return <RepeatOneIcon className={className} titleAccess='Iteration' />;
    case 'Condition':
      // Material Symbols stat_0 (Apache-2.0), retained for the conditional diamond.
      // https://github.com/google/material-design-icons/tree/master/symbols/web/stat_0
      return (
        <SvgIcon
          className={className}
          titleAccess='Conditional'
          viewBox='0 -960 960 960'
          data-testid='ConditionIcon'
        >
          <path d='M480-200 200-480l280-280 280 280-280 280Zm0-114 166-166-166-166-166 166 166 166Zm0-166Z' />
        </SvgIcon>
      );
    default:
      return <LayersIcon className={className} titleAccess='Sub-DAG' />;
  }
}
