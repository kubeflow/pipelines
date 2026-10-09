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

import { useId, useState } from 'react';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';
import { Description } from 'src/components/Description';
import { Button } from 'src/components/ui/button';
import { formatDateString, sanitizeExternalHref } from 'src/lib/Utils';
import 'src/components/pipelines/Pipelines.css';

interface PipelineVersionCardProps {
  pipeline: V2beta1Pipeline | null;
  selectedVersion: V2beta1PipelineVersion | undefined;
  versions: V2beta1PipelineVersion[];
  handleVersionSelected: (versionId: string) => Promise<void>;
}

export function PipelineVersionCard({
  pipeline,
  selectedVersion,
  versions,
  handleVersionSelected,
}: PipelineVersionCardProps) {
  const [summaryShown, setSummaryShown] = useState(false);
  const id = useId();
  const sourceUrl = sanitizeExternalHref(selectedVersion?.code_source_url);
  return (
    <>
      {!!pipeline && summaryShown && (
        <section className='kfp-pipeline-summary' aria-label='Static Pipeline Summary'>
          <header>
            <h2>Static Pipeline Summary</h2>
            <Button variant='ghost' onClick={() => setSummaryShown(false)}>
              Hide
            </Button>
          </header>
          <dl>
            <dt>Pipeline ID</dt>
            <dd>{pipeline.pipeline_id || 'Unable to obtain Pipeline ID'}</dd>
            {versions.length > 0 && (
              <>
                <dt>
                  <label htmlFor={id}>Version</label>
                </dt>
                <dd>
                  <select
                    id={id}
                    data-testid='version_selector'
                    name='selectedVersion'
                    value={selectedVersion?.pipeline_version_id || ''}
                    onChange={(event) => handleVersionSelected(event.target.value)}
                  >
                    {!selectedVersion && <option value=''>Select a version</option>}
                    {versions.map((version) => (
                      <option key={version.pipeline_version_id} value={version.pipeline_version_id}>
                        {version.display_name || version.name}
                      </option>
                    ))}
                  </select>
                </dd>
                {sourceUrl && (
                  <dd>
                    <a href={sourceUrl} target='_blank' rel='noopener noreferrer'>
                      Version source
                    </a>
                  </dd>
                )}
              </>
            )}
            <dt>Uploaded on</dt>
            <dd>
              {formatDateString(selectedVersion ? selectedVersion.created_at : pipeline.created_at)}
            </dd>
            <dt>Pipeline Description</dt>
            <dd>
              <Description description={pipeline.description || 'empty pipeline description'} />
            </dd>
            {selectedVersion?.description && (
              <>
                <dt>Version Description</dt>
                <dd>
                  <Description description={selectedVersion.description} />
                </dd>
              </>
            )}
          </dl>
        </section>
      )}
      {!summaryShown && (
        <div className='kfp-pipeline-show-summary'>
          <Button variant='secondary' onClick={() => setSummaryShown(true)}>
            Show Summary
          </Button>
        </div>
      )}
    </>
  );
}
