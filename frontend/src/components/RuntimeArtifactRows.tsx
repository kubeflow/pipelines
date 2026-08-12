// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { InputOutputsIOArtifact } from 'src/apisv2beta1/run';
import { ArtifactPreviewValue } from 'src/components/ArtifactPreview';
import { RoutePageFactory } from 'src/components/Router';
import { commonCss } from 'src/Css';
import {
  flattenArtifactGroups,
  getArtifactDisplayName,
  getArtifactSessionInfo,
} from 'src/lib/v2/RuntimeArtifactUtils';

export function buildRuntimeArtifactRows(groups: InputOutputsIOArtifact[] | undefined) {
  const rows: Array<[string | ReactElement | undefined, ArtifactPreviewValue]> = [];
  for (const { artifact, artifactKey, group, index } of flattenArtifactGroups(groups)) {
    const displayName = getArtifactDisplayName(artifact, artifactKey, index, group.artifacts);
    const label = artifact.artifact_id ? (
      <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(artifact.artifact_id)}>
        {displayName}
      </Link>
    ) : (
      displayName
    );
    rows.push([
      label,
      artifact.uri ? { uri: artifact.uri, providerInfo: getArtifactSessionInfo(artifact) } : '',
    ]);
  }
  return rows;
}
