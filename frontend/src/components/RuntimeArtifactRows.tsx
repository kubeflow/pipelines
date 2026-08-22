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
import ArtifactPreview, {
  ArtifactPreviewProps,
  ArtifactPreviewValue,
} from 'src/components/ArtifactPreview';
import { RoutePageFactory } from 'src/components/Router';
import { commonCss } from 'src/Css';
import {
  flattenArtifactGroups,
  getArtifactDisplayName,
  getScalarMetricEntries,
  isScalarMetricArtifact,
  isClassificationMetricArtifact,
} from 'src/lib/v2/RuntimeArtifactUtils';

type RuntimeArtifactRowValue = ArtifactPreviewValue | { text: string; artifactId?: string };

export function RuntimeArtifactValue({
  value,
  ...props
}: Omit<ArtifactPreviewProps, 'value'> & { value?: RuntimeArtifactRowValue }) {
  if (value !== null && typeof value === 'object' && 'text' in value) {
    return value.artifactId ? (
      <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(value.artifactId)}>
        {value.text}
      </Link>
    ) : (
      <>{value.text}</>
    );
  }
  return <ArtifactPreview value={value} {...props} />;
}

export function buildRuntimeArtifactRows(groups: InputOutputsIOArtifact[] | undefined) {
  const rows: Array<[string | ReactElement | undefined, RuntimeArtifactRowValue]> = [];
  for (const { artifact, artifactKey, group, index } of flattenArtifactGroups(groups)) {
    const displayName = getArtifactDisplayName(artifact, artifactKey, index, group.artifacts);
    const label = artifact.artifact_id ? (
      <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(artifact.artifact_id)}>
        {displayName}
      </Link>
    ) : (
      displayName
    );
    if (isScalarMetricArtifact(artifact)) {
      const metrics = getScalarMetricEntries(artifact);
      for (const metric of metrics) {
        rows.push([
          metrics.length > 1 ? (
            <>
              {label} / {metric.name}
            </>
          ) : (
            label
          ),
          { text: metric.value },
        ]);
      }
    } else if (artifact.uri) {
      rows.push([label, { uri: artifact.uri }]);
    } else if (isClassificationMetricArtifact(artifact)) {
      rows.push([
        label,
        {
          text: artifact.artifact_id
            ? 'View classification metrics'
            : 'Classification metrics — select the artifact in the graph to view.',
          artifactId: artifact.artifact_id,
        },
      ]);
    } else {
      rows.push([label, { text: 'No artifact URI available.' }]);
    }
  }
  return rows;
}
