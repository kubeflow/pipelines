/*
 * Copyright 2018 Google LLC
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

import { ApiRun, ApiResourceType, ApiResourceReference } from '../apis/run';
import { ApiJob } from '../apis/job';


function getPipelineId(run?: ApiRun | ApiJob): string | null {
  return run && run.pipeline_spec && run.pipeline_spec.pipeline_id || null;
}

function getFirstExperimentReferenceId(run?: ApiRun | ApiJob): string | null {
  if (run) {
    const reference = getAllExperimentReferences(run)[0];
    return reference && reference.key && reference.key.id || null;
  }
  return null;
}

function getFirstExperimentReference(run?: ApiRun | ApiJob): ApiResourceReference | null {
  return run && getAllExperimentReferences(run)[0] || null;
}

function getAllExperimentReferences(run?: ApiRun | ApiJob): ApiResourceReference[] {
  return (run && run.resource_references || [])
    .filter((ref) => ref.key && ref.key.type && ref.key.type === ApiResourceType.EXPERIMENT || false);
}

export default {
  getAllExperimentReferences,
  getFirstExperimentReference,
  getFirstExperimentReferenceId,
  getPipelineId,
};
