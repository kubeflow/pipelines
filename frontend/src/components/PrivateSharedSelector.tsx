/*
 * Copyright 2023 The Kubeflow Authors
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
import { PipelineTabsHeaders } from '../pages/PrivateAndSharedPipelines';
import './modernization/PipelineForms.css';

export interface PrivateSharedSelectorProps {
  onChange: (isPrivate: boolean) => void;
  value?: boolean;
}
export enum PipelineButtonTooltips {
  PRIVATE = 'Only people who have access to this namespace will be able to view and use this pipeline.',
  SHARED = 'Everyone in your organization will be able to view and use this pipeline.',
}

export default function PrivateSharedSelector({ onChange, value }: PrivateSharedSelectorProps) {
  const id = useId();
  const [localValue, setLocalValue] = useState(true);
  const isPrivate = value ?? localValue;
  const select = (next: boolean) => {
    if (value === undefined) setLocalValue(next);
    onChange(next);
  };
  return (
    <fieldset className='kfp-pipeline-choice'>
      <legend>Select if the new pipeline will be private or shared.</legend>
      <label title={PipelineButtonTooltips.PRIVATE}>
        <input
          id='createNewPrivatePipelineBtn'
          type='radio'
          name={id}
          checked={isPrivate}
          onChange={() => select(true)}
          aria-describedby={`${id}-scope`}
        />
        {PipelineTabsHeaders.PRIVATE}
      </label>
      <label title={PipelineButtonTooltips.SHARED}>
        <input
          id='createNewSharedPipelineBtn'
          type='radio'
          name={id}
          checked={!isPrivate}
          onChange={() => select(false)}
          aria-describedby={`${id}-scope`}
        />
        {PipelineTabsHeaders.SHARED}
      </label>
      <p id={`${id}-scope`} className='kfp-pipeline-form-hint'>
        {isPrivate ? PipelineButtonTooltips.PRIVATE : PipelineButtonTooltips.SHARED}
      </p>
    </fieldset>
  );
}
