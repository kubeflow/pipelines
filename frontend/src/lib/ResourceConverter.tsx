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

import { BaseResource } from 'src/pages/ResourceSelector';
import { V2beta1Experiment } from 'src/apisv2beta1/experiment';
import { V2beta1Pipeline, V2beta1PipelineVersion } from 'src/apisv2beta1/pipeline';

export function convertExperimentToResource(e: V2beta1Experiment): BaseResource {
  return {
    id: e.experiment_id,
    name: e.display_name,
    description: e.description,
    created_at: e.created_at,
  };
}

export function convertPipelineToResource(p: V2beta1Pipeline): BaseResource {
  return {
    id: p.pipeline_id,
    name: p.display_name,
    description: p.description,
    created_at: p.created_at,
    error: p.error?.toString(),
  };
}

export function convertPipelineVersionToResource(v: V2beta1PipelineVersion): BaseResource {
  return {
    id: v.pipeline_version_id,
    name: v.display_name,
    description: v.description,
    created_at: v.created_at,
    error: v.error?.toString(),
  };
}
