/**
 * Copyright 2023 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import { Apis, BuildInfo } from '../lib/Apis';
import { logger } from './Utils';

interface BuildInfoProviderState {
  buildInfo: BuildInfo | undefined;
}

export const BuildInfoContext = React.createContext<BuildInfo | undefined>(undefined);
export class BuildInfoProvider extends React.Component<{}, BuildInfoProviderState> {
  state = {
    buildInfo: undefined,
  };
  async componentDidMount() {
    try {
      const buildInfo = await Apis.getBuildInfo();
      this.setState({ buildInfo });
    } catch (err) {
      logger.error('Failed to retrieve build info', err);
    }
  }
  render() {
    return <BuildInfoContext.Provider value={this.state.buildInfo} {...this.props} />;
  }
}
