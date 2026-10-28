/*
 * Copyright 2021 The Kubeflow Authors
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

import type { ReactNode } from 'react';
import { V2Pipeline, V2PipelineVersion } from 'src/apisv2/pipeline';
import { Description } from 'src/components/Description';
import { InspectionFields } from 'src/components/inspection/InspectionFields';
import { formatDateString, sanitizeExternalHref } from 'src/lib/Utils';

interface PipelineVersionCardProps {
  pipeline: V2Pipeline | null;
  selectedVersion: V2PipelineVersion | undefined;
}

const renderValue = ({ value }: { value?: ReactNode }) => <>{value}</>;

export function PipelineVersionCard({ pipeline, selectedVersion }: PipelineVersionCardProps) {
  if (!pipeline) return null;
  const sourceUrl = sanitizeExternalHref(selectedVersion?.code_source_url);
  return (
    <div className='kfp-detail-sections'>
      <InspectionFields<ReactNode>
        title='Pipeline details'
        valueComponent={renderValue}
        fields={[
          ['Pipeline ID', pipeline.pipeline_id || 'Unavailable'],
          ['Uploaded on', formatDateString(pipeline.created_at)],
          ['Description', <Description description={pipeline.description || 'No description'} />],
        ]}
      />
      {selectedVersion && (
        <InspectionFields<ReactNode>
          title='Version details'
          valueComponent={renderValue}
          fields={[
            ['Version', selectedVersion.display_name || selectedVersion.name || 'Unnamed version'],
            ['Version ID', selectedVersion.pipeline_version_id || 'Unavailable'],
            ['Uploaded on', formatDateString(selectedVersion.created_at)],
            [
              'Description',
              <Description description={selectedVersion.description || 'No description'} />,
            ],
            ...(sourceUrl
              ? [
                  [
                    'Source',
                    <a href={sourceUrl} target='_blank' rel='noopener noreferrer'>
                      Version source
                    </a>,
                  ] as [string, ReactNode],
                ]
              : []),
          ]}
        />
      )}
    </div>
  );
}
