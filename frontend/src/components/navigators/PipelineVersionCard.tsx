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

import Button from '@material-ui/core/Button';
import FormControl from '@material-ui/core/FormControl';
import InputLabel from '@material-ui/core/InputLabel';
import MenuItem from '@material-ui/core/MenuItem';
import Paper from '@material-ui/core/Paper';
import Select from '@material-ui/core/Select';
import React, { useState } from 'react';
import { ApiPipeline, ApiPipelineVersion } from 'src/apis/pipeline';
import { Description } from 'src/components/Description';
import { commonCss } from 'src/Css';
import { formatDateString } from 'src/lib/Utils';

interface PipelineVersionCardProps {
  apiPipeline: ApiPipeline | null;
  selectedVersion: ApiPipelineVersion | undefined;
  versions: ApiPipelineVersion[];
  handleVersionSelected: (versionId: string) => Promise<void>;
}

export function PipelineVersionCard({
  apiPipeline,
  selectedVersion,
  versions,
  handleVersionSelected,
}: PipelineVersionCardProps) {
  const [summaryShown, setSummaryShown] = useState(false);

  const createVersionUrl = () => {
    return selectedVersion?.code_source_url;
  };

  return (
    <>
      {!!apiPipeline && summaryShown && (
        <Paper className='absolute bottom-3 left-20 p-5 w-136 z-20'>
          <div className='items-baseline flex justify-between'>
            <div className={commonCss.header}>Static Pipeline Summary</div>
            <Button onClick={() => setSummaryShown(false)} color='secondary'>
              Hide
            </Button>
          </div>
          <div className='text-gray-900 mt-5'>Pipeline ID</div>
          <div>{apiPipeline.id || 'Unable to obtain Pipeline ID'}</div>
          {versions.length > 0 && (
            <>
              <div className='text-gray-900 mt-5'>
                <form autoComplete='off'>
                  <FormControl>
                    <InputLabel>Version</InputLabel>
                    <Select
                      aria-label='version_selector'
                      data-testid='version_selector'
                      value={
                        selectedVersion ? selectedVersion.id : apiPipeline.default_version!.id!
                      }
                      onChange={event => handleVersionSelected(event.target.value)}
                      inputProps={{ id: 'version-selector', name: 'selectedVersion' }}
                    >
                      {versions.map((v, _) => (
                        <MenuItem key={v.id} value={v.id}>
                          {v.name}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </form>
              </div>
              <div className='text-blue-500 mt-5'>
                <a href={createVersionUrl()} target='_blank' rel='noopener noreferrer'>
                  Version source
                </a>
              </div>
            </>
          )}
          <div className='text-gray-900 mt-5'>Uploaded on</div>
          <div>
            {selectedVersion
              ? formatDateString(selectedVersion.created_at)
              : formatDateString(apiPipeline.created_at)}
          </div>

          <div className='text-gray-900 mt-5'>Pipeline Description</div>
          <Description description={apiPipeline.description || 'empty pipeline description'} />

          {/* selectedVersion is always populated by either selected or pipeline default version if it exists */}
          {selectedVersion && selectedVersion.description ? (
            <>
              <div className='text-gray-900 mt-5'>
                {selectedVersion.id === apiPipeline.default_version?.id
                  ? 'Default Version Description'
                  : 'Version Description'}
              </div>
              <Description description={selectedVersion.description} />
            </>
          ) : null}
        </Paper>
      )}
      {!summaryShown && (
        <div className='flex absolute bottom-5 left-10 pb-5 pl-10 bg-transparent z-20'>
          <Button onClick={() => setSummaryShown(!summaryShown)} color='secondary'>
            Show Summary
          </Button>
        </div>
      )}
    </>
  );
}
