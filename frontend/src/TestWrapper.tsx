// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
import type { PropsWithChildren } from 'react';
import { StyledEngineProvider, ThemeProvider } from '@mui/material/styles';
import { theme } from './Css';

export function CommonTestWrapper({ children }: PropsWithChildren) {
  return (
    <StyledEngineProvider injectFirst>
      <ThemeProvider theme={theme}>{children}</ThemeProvider>
    </StyledEngineProvider>
  );
}
