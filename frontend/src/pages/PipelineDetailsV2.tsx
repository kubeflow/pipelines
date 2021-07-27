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
import React from 'react';

interface PipelineDetailsV2Props {}

export const PipelineDetailsV2: React.FC<PipelineDetailsV2Props> = props => {
  const samples = ['1', '2'];

  const handleSelectChange = (event: React.ChangeEvent<HTMLSelectElement>) => {};
  return (
    <div style={{ display: 'flex', flexFlow: 'row wrap', justifyContent: 'flex-start' }}>
      <div style={{ margin: 'auto' }}>
        <select onChange={handleSelectChange}>
          {samples.map(sample => {
            return <option value={sample}>{sample}</option>;
          })}
        </select>
      </div>
      <div style={{ margin: 'auto' }}>
        <PipelineIRDialog onSubmit={onSubmit}></PipelineIRDialog>
      </div>
    </div>
  );
};
