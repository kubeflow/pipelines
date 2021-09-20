/**
 * Copyright 2021 The Kubeflow Authors
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

import React, { FC, useState, useEffect } from 'react';
import { logger, extendError } from './Utils';
import { Apis } from './Apis';

export interface GkeMetadata {
  projectId?: string;
  clusterName?: string;
}
export const GkeMetadataContext = React.createContext<GkeMetadata>({});
export const GkeMetadataProvider: FC<{}> = props => {
  const [metadata, setMetadata] = useState({});
  useEffect(() => {
    Apis.getClusterName()
      .then(clusterName => setMetadata(metadata => ({ ...metadata, clusterName })))
      .catch(err => logger.warn(extendError(err, 'Failed getting GKE cluster name')));
    Apis.getProjectId()
      .then(projectId => setMetadata(metadata => ({ ...metadata, projectId })))
      .catch(err => logger.warn(extendError(err, 'Failed getting GKE project ID')));
  }, []);
  return <GkeMetadataContext.Provider value={metadata} {...props} />;
};
