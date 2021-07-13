/*
 * Copyright 2018,2021 The Kubeflow Authors
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

import { Artifact, Value } from 'src/third_party/mlmd';
import { ArtifactCustomProperties, ArtifactProperties } from './Api';

export const doubleValue = (number: number) => {
  const value = new Value();
  value.setDoubleValue(number);
  return value;
};

export const intValue = (number: number) => {
  const value = new Value();
  value.setIntValue(number);
  return value;
};

export const stringValue = (string: String) => {
  const value = new Value();
  value.setStringValue(String(string));
  return value;
};

export const buildTestModel = () => {
  const model = new Artifact();
  model.setId(1);
  model.setTypeId(1);
  model.setUri('gs://my-bucket/mnist');
  model.getPropertiesMap().set(ArtifactProperties.NAME, stringValue('test model'));
  model.getPropertiesMap().set(ArtifactProperties.DESCRIPTION, stringValue('A really great model'));
  model.getPropertiesMap().set(ArtifactProperties.VERSION, stringValue('v1'));
  model
    .getPropertiesMap()
    .set(ArtifactProperties.CREATE_TIME, stringValue('2019-06-12T01:21:48.259263Z'));
  model
    .getPropertiesMap()
    .set(
      ArtifactProperties.ALL_META,
      stringValue(
        '{"hyperparameters": {"early_stop": true, ' +
          '"layers": [10, 3, 1], "learning_rate": 0.5}, ' +
          '"model_type": "neural network", ' +
          '"training_framework": {"name": "tensorflow", "version": "v1.0"}}',
      ),
    );
  model
    .getCustomPropertiesMap()
    .set(ArtifactCustomProperties.WORKSPACE, stringValue('workspace-1'));
  model.getCustomPropertiesMap().set(ArtifactCustomProperties.RUN, stringValue('1'));
  return model;
};

export const testModel = buildTestModel();
