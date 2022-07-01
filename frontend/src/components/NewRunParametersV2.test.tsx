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
            intParam: {
              parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
              defaultValue: 123,
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
    expect(screen.getByText('intParam - NUMBER_INTEGER'));
    expect(screen.getByDisplayValue('123'));
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
            boolParam: {
              parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
              defaultValue: true,
            },
          }}
        ></NewRunParametersV2>
      </CommonTestWrapper>,
    );

    const strParam = screen.getByDisplayValue('string value');
    fireEvent.change(strParam, { target: { value: 'new string' } });
    expect(strParam.closest('input').value).toEqual('new string');

    const intParam = screen.getByDisplayValue('123');
    fireEvent.change(intParam, { target: { value: 999 } });
    expect(intParam.closest('input').value).toEqual('999');

    const boolParam = screen.getByDisplayValue('true');
    fireEvent.change(boolParam, { target: { value: false } });
    expect(boolParam.closest('input').value).toEqual('false');
  });

  it('test convertInput function for string type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        strParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'string value',
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const strParam = screen.getByDisplayValue('string value');
    fireEvent.change(strParam, { target: { value: 'new string' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      strParam: 'new string',
    });
  });

  it('test convertInput function for string type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        strParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const strParam = screen.getByLabelText('strParam - STRING');
    fireEvent.change(strParam, { target: { value: 'new string' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      strParam: 'new string',
    });
  });

  it('test convertInput function for boolean type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
          defaultValue: true,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const boolParam = screen.getByDisplayValue('true');
    fireEvent.change(boolParam, { target: { value: 'false' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: false,
    });
  });

  it('test convertInput function for boolean type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const boolParam = screen.getByLabelText('boolParam - BOOLEAN');
    fireEvent.change(boolParam, { target: { value: 'true' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: true,
    });
  });

  it('test convertInput function for boolean type with invalid input (Uppercase)', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const boolParam = screen.getByLabelText('boolParam - BOOLEAN');
    fireEvent.change(boolParam, { target: { value: 'True' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: null,
    });
  });

  it('test convertInput function for integer type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 123,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const intParam = screen.getByDisplayValue('123');
    fireEvent.change(intParam, { target: { value: '456' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: 456,
    });
  });

  it('test convertInput function for integer type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const intParam = screen.getByLabelText('intParam - NUMBER_INTEGER');
    fireEvent.change(intParam, { target: { value: '789' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: 789,
    });
  });

  it('test convertInput function for integer type with invalid input (float)', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'defalut pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const intParam = screen.getByLabelText('intParam - NUMBER_INTEGER');
    fireEvent.change(intParam, { target: { value: '7.89' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: null,
    });
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
