// Copyright 2026 The Kubeflow Authors. Licensed under the Apache License, Version 2.0.
import 'src/build/tailwind.output.css';
import '@xyflow/react/dist/style.css';
import { StyledEngineProvider, ThemeProvider } from '@mui/material/styles';
import { createRoot } from 'react-dom/client';
import { config as aceConfig } from 'ace-builds';
import { cssRule } from 'typestyle';
import { fonts, theme } from 'src/Css';
import { Viewer, type ViewerState } from './Viewer';
import './viewer.css';

// Match the application's root sizing: graph cards and layout both use rem units.
cssRule('html, body, #root', {
  background: 'white',
  color: 'rgba(0, 0, 0, .66)',
  display: 'flex',
  fontFamily: fonts.main,
  fontSize: 13,
  height: '100%',
  width: '100%',
});

// The shared inspector can display JSON schemas; offline read-only views need no worker.
aceConfig.setDefaultValue('session', 'useWorker', false);

const { state, live }: { state: ViewerState; live: boolean } = JSON.parse(
  document.getElementById('kfp-data')!.textContent!,
);
createRoot(document.getElementById('root')!).render(
  <StyledEngineProvider injectFirst>
    <ThemeProvider theme={theme}>
      <Viewer initial={state} live={live} />
    </ThemeProvider>
  </StyledEngineProvider>,
);
