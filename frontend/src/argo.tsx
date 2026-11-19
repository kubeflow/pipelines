// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import 'src/build/tailwind.output.css';
import '@xyflow/react/dist/style.css';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { StyledEngineProvider, ThemeProvider } from '@mui/material/styles';
import { cssRule } from 'typestyle';
import { fonts, theme } from 'src/Css';
import ArgoWorkspace from 'src/pages/ArgoWorkspace';

cssRule('html, body, #root', {
  margin: 0,
  background: 'white',
  color: 'rgba(0, 0, 0, .66)',
  fontFamily: fonts.main,
  fontSize: 13,
  height: '100%',
  width: '100%',
});

const namespace = new URLSearchParams(window.location.search).get('namespace') || 'argo';
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={new QueryClient()}>
      <StyledEngineProvider injectFirst>
        <ThemeProvider theme={theme}>
          <ArgoWorkspace key={namespace} namespace={namespace} />
        </ThemeProvider>
      </StyledEngineProvider>
    </QueryClientProvider>
  </StrictMode>,
);
