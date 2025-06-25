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
// React Flow CSS (required for v11+)
import 'reactflow/dist/style.css';
import * as React from 'react';
import * as ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider } from 'react-query';
import { HashRouter } from 'react-router-dom';
import { cssRule } from 'typestyle';
import Router from './components/Router';
import { fonts, theme } from './Css';
import { initFeatures } from './features';
import { Deployments, KFP_FLAGS } from './lib/Flags';
import { GkeMetadataProvider } from './lib/GkeMetadata';
import {
  init as initKfClient,
  NamespaceContext,
  NamespaceContextProvider,
} from './lib/KubeflowClient';
import { BuildInfoProvider } from './lib/BuildInfo';
import { ThemeProvider, Theme, StyledEngineProvider } from '@mui/material/styles';

declare module '@mui/styles/defaultTheme' {
  // eslint-disable-next-line @typescript-eslint/no-empty-interface
  interface DefaultTheme extends Theme {}
}

// import { ReactQueryDevtools } from 'react-query/devtools';

// TODO: license headers

if (KFP_FLAGS.DEPLOYMENT === Deployments.KUBEFLOW) {
  initKfClient();
}

cssRule('html, body, #root', {
  background: 'white',
  color: 'rgba(0, 0, 0, .66)',
  display: 'flex',
  fontFamily: fonts.main,
  fontSize: 13,
  height: '100%',
  width: '100%',
});

initFeatures();

export const queryClient = new QueryClient();
const app = (
  <QueryClientProvider client={queryClient}>
    <StyledEngineProvider injectFirst>
      <ThemeProvider theme={theme}>
        <BuildInfoProvider>
          <GkeMetadataProvider>
            <HashRouter>
              <Router />
            </HashRouter>
          </GkeMetadataProvider>
        </BuildInfoProvider>
      </ThemeProvider>
    </StyledEngineProvider>
    {/* <ReactQueryDevtools initialIsOpen={false} /> */}
  </QueryClientProvider>
);
const container = document.getElementById('root');
const root = ReactDOM.createRoot(container!);
root.render(
  KFP_FLAGS.DEPLOYMENT === Deployments.KUBEFLOW ? (
    <NamespaceContextProvider>{app}</NamespaceContextProvider>
  ) : (
    <NamespaceContext.Provider value={process.env.REACT_APP_NAMESPACE || undefined}>
      {app}
    </NamespaceContext.Provider>
  ),
);
