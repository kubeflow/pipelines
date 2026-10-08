// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import SvgIcon from '@mui/material/SvgIcon';

// Material Symbols Outlined (Apache-2.0):
// https://github.com/google/material-design-icons/tree/master/symbols/web
const paths: Record<string, string> = {
  'Sub-DAG':
    'M600-120v-120H440v-400h-80v120H80v-320h280v120h240v-120h280v320H600v-120h-80v320h80v-120h280v320H600ZM160-760v160-160Zm520 400v160-160Zm0-400v160-160Zm0 160h120v-160H680v160Zm0 400h120v-160H680v160ZM160-600h120v-160H160v160Z',
  Loop: 'M204-318q-22-38-33-78t-11-82q0-134 93-228t227-94h7l-64-64 56-56 160 160-160 160-56-56 64-64h-7q-100 0-170 70.5T240-478q0 26 6 51t18 49l-60 60ZM481-40 321-200l160-160 56 56-64 64h7q100 0 170-70.5T720-482q0-26-6-51t-18-49l60-60q22 38 33 78t11 82q0 134-93 228t-227 94h-7l64 64-56 56Z',
  Iteration:
    'm480-320 160-160-160-160-56 56 64 64H320v80h168l-64 64 56 56Zm0 240q-83 0-156-31.5T197-197q-54-54-85.5-127T80-480q0-83 31.5-156T197-763q54-54 127-85.5T480-880q83 0 156 31.5T763-763q54 54 85.5 127T880-480q0 83-31.5 156T763-197q-54 54-127 85.5T480-80Zm0-80q134 0 227-93t93-227q0-134-93-227t-227-93q-134 0-227 93t-93 227q0 134 93 227t227 93Zm0-320Z',
  Condition:
    'M480-200 200-480l280-280 280 280-280 280Zm0-114 166-166-166-166-166 166 166 166Zm0-166Z',
};

export default function SubDagTypeIcon({ kind }: { kind: string }) {
  return (
    <SvgIcon className='text-mui-blue-600' viewBox='0 -960 960 960' titleAccess={kind}>
      <path d={paths[kind] || paths['Sub-DAG']} />
    </SvgIcon>
  );
}
