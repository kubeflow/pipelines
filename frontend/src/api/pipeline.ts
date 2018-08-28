// Copyright 2018 Google LLC
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

import { Parameter } from './parameter';

export class Pipeline {
  public id: string;
  // TODO: add author. Waiting for (#896)
  // TODO: add version. Waiting for (#166)
  // TODO: this will eventually be renamed "uploaded_on" (#983)
  public created_at: string;
  public name: string;
  public description?: string;
  public parameters: Parameter[];

  constructor() {
    this.id = '';
    this.name = '';
    this.description = '';
    this.parameters = [];
    this.created_at = '';
  }

  public static buildFromObject(pipeline: any): Pipeline {
    const newPipeline = new Pipeline();
    newPipeline.id = pipeline.id;
    newPipeline.name = pipeline.name;
    newPipeline.description = pipeline.description;
    newPipeline.parameters = pipeline.parameters;
    newPipeline.created_at = pipeline.created_at;
    return newPipeline;
  }
}

export interface PipelineTemplate {
  template: string;
}
