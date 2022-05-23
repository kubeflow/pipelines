/*
 * Copyright 2022 The Kubeflow Authors
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

import 'jest';
import React from 'react';
import { testBestPractices } from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { ParameterType_ParameterTypeEnum } from 'src/generated/pipeline_spec/pipeline_spec';
import NewRunParametersV2 from 'src/components/NewRunParametersV2';

testBestPractices();

describe('NewRunParametersV2', () => {
  it('shows parameters', () => {
    render(
      <CommonTestWrapper>
        <NewRunParametersV2
          titleMessage='Specify parameters required by the pipeline'
          specParameters={{
            strParam: {
              parameterType: ParameterType_ParameterTypeEnum.STRING,
              defaultValue: 'string value',
            },
            boolParam: {
              parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
              defaultValue: true,
            },
          }}
        ></NewRunParametersV2>
      </CommonTestWrapper>,
    );

    expect(screen.getByText('Run parameters'));
    expect(screen.getByText('Specify parameters required by the pipeline'));
    expect(screen.getByText('strParam - STRING'));
    expect(screen.getByDisplayValue('string value'));
    expect(screen.getByText('boolParam - BOOLEAN'));
    expect(screen.getByDisplayValue('true'));
  });

  it('edits parameters', () => {
    render(
      <CommonTestWrapper>
        <NewRunParametersV2
          titleMessage='Specify parameters required by the pipeline'
          specParameters={{
            strParam: {
              parameterType: ParameterType_ParameterTypeEnum.STRING,
              defaultValue: 'string value',
            },
            intParam: {
              parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
              defaultValue: 123,
            },
          }}
        ></NewRunParametersV2>
      </CommonTestWrapper>,
    );

    const strParam = screen.getByDisplayValue('string value');
    fireEvent.change(strParam, { target: { value: 'new string' } });
    expect(strParam.closest('input').value).toEqual('new string');

    const intParam = screen.getByDisplayValue(123);
    fireEvent.change(intParam, { target: { value: 999 } });
    expect(intParam.closest('input').value).toEqual('999');
  });

  it('does not display any text fields if there are no parameters', () => {
    const { container } = render(
      <CommonTestWrapper>
        <NewRunParametersV2
          titleMessage='Specify parameters required by the pipeline'
          specParameters={{}}
        ></NewRunParametersV2>
      </CommonTestWrapper>,
    );

    expect(container.querySelector('input').type).toEqual('checkbox');
  });
});
