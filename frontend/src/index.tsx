/*
 * Copyright 2018 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// import './CSSReset';
import 'src/build/tailwind.output.css';
import * as React from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from 'react-query';
import { HashRouter } from 'react-router-dom';
import { cssRule } from 'typestyle';
import Router from './components/Router';
import { Deployments, KFP_FLAGS } from './lib/Flags';
import { GkeMetadataProvider } from './lib/GkeMetadata';
import {
  init as initKfClient,
  NamespaceContext,
  NamespaceContextProvider,
} from './lib/KubeflowClient';
import { BuildInfoProvider } from './lib/BuildInfo';
import { theme } from './Css';
import { ThemeProvider, CssBaseline } from '@mui/material/styles';

// TODO: license headers

if (KFP_FLAGS.DEPLOYMENT === Deployments.KUBEFLOW) {
  initKfClient();
}

cssRule('html, body, #root', {
  height: '100%',
  width: '100%',
  margin: 0,
  padding: 0,
});

initFeatures();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: false,
    },
  },
});

const app = (
  <ThemeProvider theme={theme}>
    <CssBaseline />
    <HashRouter>
      <QueryClientProvider client={queryClient}>
        <BuildInfoProvider>
          <GkeMetadataProvider>
            <Router />
          </GkeMetadataProvider>
        </BuildInfoProvider>
      </QueryClientProvider>
    </HashRouter>
  </ThemeProvider>
);

const container = document.getElementById('root');
if (!container) throw new Error('Failed to find the root element');
const root = createRoot(container);

root.render(
  KFP_FLAGS.DEPLOYMENT === Deployments.KUBEFLOW ? (
    <React.StrictMode>
      <NamespaceContextProvider>{app}</NamespaceContextProvider>
    </React.StrictMode>
  ) : (
    <React.StrictMode>
      <NamespaceContext.Provider value={undefined}>{app}</NamespaceContext.Provider>
    </React.StrictMode>
  )
);
