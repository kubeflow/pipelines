// Copyright 2026 The Kubeflow Authors. Licensed under the Apache License, Version 2.0.
import 'src/build/tailwind.output.css';
import '@xyflow/react/dist/style.css';
import { createRoot } from 'react-dom/client';
import 'src/index.css';
import { Viewer, type ViewerState } from './Viewer';
import './viewer.css';

const { state, live }: { state: ViewerState; live: boolean } = JSON.parse(
  document.getElementById('kfp-data')!.textContent!,
);
createRoot(document.getElementById('root')!).render(<Viewer initial={state} live={live} />);
