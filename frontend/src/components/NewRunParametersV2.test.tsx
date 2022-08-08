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
    const props = {
      titleMessage: 'Specify parameters required by the pipeline',
      specParameters: {
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
      },
      clonedRuntimeConfig: {},
    };
    render(<NewRunParametersV2 {...props} />);

    screen.getByText('Run parameters');
    screen.getByText('Specify parameters required by the pipeline');
    screen.getByText('strParam - STRING');
    screen.getByDisplayValue('string value');
    screen.getByText('boolParam - BOOLEAN');
    screen.getByDisplayValue('true');
    screen.getByText('intParam - NUMBER_INTEGER');
    screen.getByDisplayValue('123');
  });

  it('edits parameters', () => {
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
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
      },
      clonedRuntimeConfig: {},
    };
    render(<NewRunParametersV2 {...props} />);

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

  it('call convertInput function for string type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        strParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'string value',
        },
      },
      clonedRuntimeConfig: {},
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
    screen.getByDisplayValue('new string');
  });

  it('call convertInput function for string type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        strParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const strParam = screen.getByLabelText('strParam - STRING');
    fireEvent.change(strParam, { target: { value: 'new string' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      strParam: 'new string',
    });
    screen.getByDisplayValue('new string');
  });

  it('call convertInput function for boolean type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
          defaultValue: true,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByDisplayValue('true');
    fireEvent.change(boolParam, { target: { value: 'false' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: false,
    });
    screen.getByDisplayValue('false');
  });

  it('call convertInput function for boolean type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByLabelText('boolParam - BOOLEAN');
    fireEvent.change(boolParam, { target: { value: 'true' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: true,
    });
    screen.getByDisplayValue('true');
  });

  it('call convertInput function for boolean type with invalid input (Uppercase)', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByLabelText('boolParam - BOOLEAN');
    fireEvent.change(boolParam, { target: { value: 'True' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: null,
    });
    screen.getByDisplayValue('True');
  });

  it('call convertInput function for integer type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 123,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByDisplayValue('123');
    fireEvent.change(intParam, { target: { value: '456' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: 456,
    });
    screen.getByDisplayValue('456');
  });

  it('call convertInput function for integer type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByLabelText('intParam - NUMBER_INTEGER');
    fireEvent.change(intParam, { target: { value: '789' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: 789,
    });
    screen.getByDisplayValue('789');
  });

  it('call convertInput function for integer type with invalid input (float)', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByLabelText('intParam - NUMBER_INTEGER');
    fireEvent.change(intParam, { target: { value: '7.89' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: null,
    });
    screen.getByDisplayValue('7.89');
  });

  it('call convertInput function for double type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        doubleParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_DOUBLE,
          defaultValue: 1.23,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const doubleParam = screen.getByDisplayValue('1.23');
    fireEvent.change(doubleParam, { target: { value: '4.56' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      doubleParam: 4.56,
    });
    screen.getByDisplayValue('4.56');
  });

  it('call convertInput function for double type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        doubleParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_DOUBLE,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const doubleParam = screen.getByLabelText('doubleParam - NUMBER_DOUBLE');
    fireEvent.change(doubleParam, { target: { value: '7.89' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      doubleParam: 7.89,
    });
    screen.getByDisplayValue('7.89');
  });

  it('call convertInput function for LIST type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
          defaultValue: [1, 2, 3],
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByDisplayValue('[1,2,3]');
    fireEvent.change(listParam, { target: { value: '[4,5,6]' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      listParam: [4, 5, 6],
    });
    screen.getByDisplayValue('[4,5,6]');
  });

  it('call convertInput function for LIST type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByLabelText('listParam - LIST');
    fireEvent.change(listParam, { target: { value: '[4,5,6]' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      listParam: [4, 5, 6],
    });
    screen.getByDisplayValue('[4,5,6]');
  });

  it('call convertInput function for LIST type with invalid input (invalid JSON form)', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByLabelText('listParam - LIST');
    fireEvent.change(listParam, { target: { value: '[4,5,6' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      listParam: null,
    });
    screen.getByDisplayValue('[4,5,6');
  });

  it('call convertInput function for STRUCT type with default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
          defaultValue: { A: 1, B: 2 },
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByDisplayValue('{"A":1,"B":2}');
    fireEvent.change(structParam, { target: { value: '{"C":3,"D":4}' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      structParam: { C: 3, D: 4 },
    });
    screen.getByDisplayValue('{"C":3,"D":4}');
  });

  it('call convertInput function for STRUCT type without default value', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - STRUCT');
    fireEvent.change(structParam, { target: { value: '{"A":1,"B":2}' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      structParam: { A: 1, B: 2 },
    });
    screen.getByDisplayValue('{"A":1,"B":2}');
  });

  it('call convertInput function for STRUCT type with invalid input (invalid JSON form)', () => {
    const handleParameterChangeSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - STRUCT');
    fireEvent.change(structParam, { target: { value: '"A":1,"B":2' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      structParam: null,
    });
    screen.getByDisplayValue('"A":1,"B":2');
  });

  it('set input as valid type with valid default integer input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 123,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    expect(setIsValidInputSpy).toHaveBeenCalledTimes(1);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(true);
    screen.getByDisplayValue('123');
  });

  it('set input as invalid type with no default integer input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    expect(setIsValidInputSpy).toHaveBeenCalledTimes(1);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
  });

  it('show error message for invalid integer input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByLabelText('intParam - NUMBER_INTEGER');
    fireEvent.change(intParam, { target: { value: '123b' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123b');
    screen.getByText('Invalid input. This parameter should be in NUMBER_INTEGER type');
  });

  it('show error message for missing integer input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 123,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByDisplayValue('123');
    fireEvent.change(intParam, { target: { value: '' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByText('Missing parameter.');
  });

  it('show error message for invalid boolean input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByLabelText('boolParam - BOOLEAN');
    fireEvent.change(boolParam, { target: { value: '123' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123');
    screen.getByText('Invalid input. This parameter should be in BOOLEAN type');
  });

  it('set input as valid type with valid boolean input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByLabelText('boolParam - BOOLEAN');
    fireEvent.change(boolParam, { target: { value: 'true' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(true);
    screen.getByDisplayValue('true');
  });

  it('show error message for invalid double input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        doubleParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_DOUBLE,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const doubleParam = screen.getByLabelText('doubleParam - NUMBER_DOUBLE');
    fireEvent.change(doubleParam, { target: { value: '123b' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123b');
    screen.getByText('Invalid input. This parameter should be in NUMBER_DOUBLE type');
  });

  it('show error message for invalid list input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByLabelText('listParam - LIST');
    fireEvent.change(listParam, { target: { value: '123' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123');
    screen.getByText('Invalid input. This parameter should be in LIST type');
  });

  it('show error message for invalid list input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByLabelText('listParam - LIST');
    fireEvent.change(listParam, { target: { value: '[1,2,3' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('[1,2,3');
    screen.getByText('Invalid input. This parameter should be in LIST type');
  });

  it('show error message for invalid struct input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - STRUCT');
    fireEvent.change(structParam, { target: { value: '123' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123');
    screen.getByText('Invalid input. This parameter should be in STRUCT type');
  });

  it('show error message for invalid struct input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - STRUCT');
    fireEvent.change(structParam, { target: { value: '[1,2,3]' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('[1,2,3]');
    screen.getByText('Invalid input. This parameter should be in STRUCT type');
  });

  it('set input as valid type with valid struct input', () => {
    const setIsValidInputSpy = jest.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - STRUCT');
    fireEvent.change(structParam, { target: { value: '{"A":1,"B":2}' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(true);
    screen.getByDisplayValue('{"A":1,"B":2}');
  });

  it('shows parameters from cloned RuntimeConfig', () => {
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
        strParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
        },
      },
      clonedRuntimeConfig: { parameters: { intParam: 123, strParam: 'string_value' } },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: jest.fn(),
    };
    render(<NewRunParametersV2 {...props} />);

    screen.getByDisplayValue('123');
    screen.getByDisplayValue('string_value');
  });

  it('edits parameters filled by cloned RuntimeConfig', () => {
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
        strParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
        },
      },
      clonedRuntimeConfig: { parameters: { intParam: 123, strParam: 'string_value' } },
      handlePipelineRootChange: jest.fn(),
      handleParameterChange: jest.fn(),
      setIsValidInput: jest.fn(),
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByDisplayValue('123');
    fireEvent.change(intParam, { target: { value: '456' } });
    screen.getByDisplayValue('456');
    const strParam = screen.getByDisplayValue('string_value');
    fireEvent.change(strParam, { target: { value: 'new_string' } });
    screen.getByDisplayValue('new_string');
  });

  it('does not display any text fields if there are no parameters', () => {
    const { container } = render(
      <CommonTestWrapper>
        <NewRunParametersV2
          titleMessage='Specify parameters required by the pipeline'
          specParameters={{}}
          clonedRuntimeConfig={{}}
        ></NewRunParametersV2>
      </CommonTestWrapper>,
    );

    expect(container.querySelector('input').type).toEqual('checkbox');
  });
});
