/*
 * Copyright 2018 Google LLC
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

import './CSSReset';
import * as React from 'react';
import * as ReactDOM from 'react-dom';
import MuiThemeProvider from '@material-ui/core/styles/MuiThemeProvider';
import Router from './components/Router';
import { cssRule } from 'typestyle';
import { theme, fonts } from './Css';

// TODO: license headers

cssRule('html, body, #root', {
  background: 'white',
  color: 'rgba(0, 0, 0, .66)',
  display: 'flex',
  fontFamily: fonts.main,
  fontSize: 13,
  height: '100%',
  width: '100%',
});

ReactDOM.render(
  <MuiThemeProvider theme={theme}>
    <Router />
  </MuiThemeProvider>,
  document.getElementById('root'),
);
