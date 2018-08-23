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

export class RunMetadata {
  public id = '';
  public name = '';
  public namespace = '';
  public created_at = '';
  public scheduled_at = '';
  public status = '';

  public static buildFromObject(run: any): RunMetadata {
    const newRunMetadata = new RunMetadata();
    newRunMetadata.id = run.id;
    newRunMetadata.name = run.name;
    newRunMetadata.namespace = run.namespace;
    newRunMetadata.created_at = run.created_at;
    newRunMetadata.scheduled_at = run.scheduled_at;
    newRunMetadata.status = run.status;
    return newRunMetadata;
  }
}

export class Run {
  public run = RunMetadata.buildFromObject({});
  public workflow = '';

  public static buildFromObject(oldRun: any): Run {
    const newRun = new Run();
    newRun.run = RunMetadata.buildFromObject(oldRun.run);
    newRun.workflow = oldRun.workflow;
    return newRun;
  }
}
