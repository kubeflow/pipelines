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

import { render, screen } from '@testing-library/react';
import React from 'react';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { Artifact, Value } from 'src/third_party/mlmd';
import { ArtifactTitle } from './ArtifactTitle';

testBestPractices();
describe('ArtifactTitle', () => {
  const artifact = new Artifact();
  const artifactName = 'fake-artifact';
  const artifactId = 123;
  beforeEach(() => {
    artifact.setId(artifactId);
    artifact.getCustomPropertiesMap().set('display_name', new Value().setStringValue(artifactName));
  });

  it('Shows artifact name', () => {
    render(
      <CommonTestWrapper>
        <ArtifactTitle artifact={artifact}></ArtifactTitle>
      </CommonTestWrapper>,
    );
    screen.getByText(artifactName, { selector: 'a', exact: false });
  });

  it('Shows artifact description', () => {
    render(
      <CommonTestWrapper>
        <ArtifactTitle artifact={artifact}></ArtifactTitle>
      </CommonTestWrapper>,
    );
    screen.getByText(/This step corresponds to artifact/);
  });
});
