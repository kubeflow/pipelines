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
    screen.getByLabelText('strParam - string');
    screen.getByDisplayValue('string value');
    screen.getByLabelText('boolParam - boolean');
    screen.getByDisplayValue('true');
    screen.getByLabelText('intParam - integer');
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
    const handleParameterChangeSpy = vi.fn();
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props}></NewRunParametersV2>);

    const strParam = screen.getByDisplayValue('string value');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(strParam, { target: { value: 'new string' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      strParam: 'new string',
    });
    screen.getByDisplayValue('new string');
  });

  it('call convertInput function for string type without default value', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        strParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const strParam = screen.getByLabelText('strParam - string');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(strParam, { target: { value: 'new string' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      strParam: 'new string',
    });
    screen.getByDisplayValue('new string');
  });

  it('call convertInput function for boolean type with default value', () => {
    const handleParameterChangeSpy = vi.fn();
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByDisplayValue('true');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(boolParam, { target: { value: 'false' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: false,
    });
    screen.getByDisplayValue('false');
  });

  it('call convertInput function for boolean type without default value', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByLabelText('boolParam - boolean');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(boolParam, { target: { value: 'true' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: true,
    });
    screen.getByDisplayValue('true');
  });

  it('call convertInput function for boolean type with invalid input (Uppercase)', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByLabelText('boolParam - boolean');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(boolParam, { target: { value: 'True' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      boolParam: null,
    });
    screen.getByDisplayValue('True');
  });

  it('call convertInput function for integer type with default value', () => {
    const handleParameterChangeSpy = vi.fn();
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    handleParameterChangeSpy.mockClear();
    const intParam = screen.getByDisplayValue('123');
    fireEvent.change(intParam, { target: { value: '456' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: 456,
    });
    screen.getByDisplayValue('456');
  });

  it('call convertInput function for integer type without default value', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByLabelText('intParam - integer');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(intParam, { target: { value: '789' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: 789,
    });
    screen.getByDisplayValue('789');
  });

  it('call convertInput function for integer type with invalid input (float)', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByLabelText('intParam - integer');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(intParam, { target: { value: '7.89' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      intParam: null,
    });
    screen.getByDisplayValue('7.89');
  });

  it('call convertInput function for double type with default value', () => {
    const handleParameterChangeSpy = vi.fn();
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const doubleParam = screen.getByDisplayValue('1.23');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(doubleParam, { target: { value: '4.56' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      doubleParam: 4.56,
    });
    screen.getByDisplayValue('4.56');
  });

  it('call convertInput function for double type without default value', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        doubleParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_DOUBLE,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const doubleParam = screen.getByLabelText('doubleParam - double');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(doubleParam, { target: { value: '7.89' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      doubleParam: 7.89,
    });
    screen.getByDisplayValue('7.89');
  });

  it('call convertInput function for LIST type with default value', () => {
    const handleParameterChangeSpy = vi.fn();
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByDisplayValue('[1,2,3]');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(listParam, { target: { value: '[4,5,6]' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      listParam: [4, 5, 6],
    });
    screen.getByDisplayValue('[4,5,6]');
  });

  it('call convertInput function for LIST type without default value', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByLabelText('listParam - list');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(listParam, { target: { value: '[4,5,6]' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      listParam: [4, 5, 6],
    });
    screen.getByDisplayValue('[4,5,6]');
  });

  it('call convertInput function for LIST type with invalid input (invalid JSON form)', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByLabelText('listParam - list');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(listParam, { target: { value: '[4,5,6' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      listParam: null,
    });
    screen.getByDisplayValue('[4,5,6');
  });

  it('call convertInput function for STRUCT type with default value', () => {
    const handleParameterChangeSpy = vi.fn();
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByDisplayValue('{"A":1,"B":2}');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(structParam, { target: { value: '{"C":3,"D":4}' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      structParam: { C: 3, D: 4 },
    });
    screen.getByDisplayValue('{"C":3,"D":4}');
  });

  it('call convertInput function for STRUCT type without default value', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - dict');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(structParam, { target: { value: '{"A":1,"B":2}' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      structParam: { A: 1, B: 2 },
    });
    screen.getByDisplayValue('{"A":1,"B":2}');
  });

  it('call convertInput function for STRUCT type with invalid input (invalid JSON form)', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - dict');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(structParam, { target: { value: '"A":1,"B":2' } });
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenLastCalledWith({
      structParam: null,
    });
    screen.getByDisplayValue('"A":1,"B":2');
  });

  it('set input as valid type with valid default integer input', () => {
    const setIsValidInputSpy = vi.fn();
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    expect(setIsValidInputSpy).toHaveBeenCalledTimes(1);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(true);
    screen.getByDisplayValue('123');
  });

  it('set input as invalid type with no default integer input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    expect(setIsValidInputSpy).toHaveBeenCalledTimes(1);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
  });

  it('show error message for invalid integer input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const intParam = screen.getByLabelText('intParam - integer');
    fireEvent.change(intParam, { target: { value: '123b' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123b');
    screen.getByText('Invalid input. This parameter should be in integer type');
  });

  it('show error message for missing integer input', () => {
    const setIsValidInputSpy = vi.fn();
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
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
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByLabelText('boolParam - boolean');
    fireEvent.change(boolParam, { target: { value: '123' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123');
    screen.getByText('Invalid input. This parameter should be in boolean type');
  });

  it('set input as valid type with valid boolean input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const boolParam = screen.getByLabelText('boolParam - boolean');
    fireEvent.change(boolParam, { target: { value: 'true' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(true);
    screen.getByDisplayValue('true');
  });

  it('show error message for invalid double input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        doubleParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_DOUBLE,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const doubleParam = screen.getByLabelText('doubleParam - double');
    fireEvent.change(doubleParam, { target: { value: '123b' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123b');
    screen.getByText('Invalid input. This parameter should be in double type');
  });

  it('show error message for invalid list input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByLabelText('listParam - list');
    fireEvent.change(listParam, { target: { value: '123' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123');
    screen.getByText('Invalid input. This parameter should be in list type');
  });

  it('show error message for invalid list input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const listParam = screen.getByLabelText('listParam - list');
    fireEvent.change(listParam, { target: { value: '[1,2,3' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('[1,2,3');
    screen.getByText('Invalid input. This parameter should be in list type');
  });

  it('show error message for invalid struct input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - dict');
    fireEvent.change(structParam, { target: { value: '123' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('123');
    screen.getByText('Invalid input. This parameter should be in dict type');
  });

  it('show error message for invalid struct input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - dict');
    fireEvent.change(structParam, { target: { value: '[1,2,3]' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(false);
    screen.getByDisplayValue('[1,2,3]');
    screen.getByText('Invalid input. This parameter should be in dict type');
  });

  it('set input as valid type with valid struct input', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    const structParam = screen.getByLabelText('structParam - dict');
    fireEvent.change(structParam, { target: { value: '{"A":1,"B":2}' } });
    expect(setIsValidInputSpy).toHaveBeenCalledTimes(2);
    expect(setIsValidInputSpy).toHaveBeenLastCalledWith(true);
    screen.getByDisplayValue('{"A":1,"B":2}');
  });

  it('show pipeline root from cloned RuntimeConfig', () => {
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
      clonedRuntimeConfig: {
        parameters: { intParam: 123, strParam: 'string_value' },
        pipeline_root: 'gs://dummy_pipeline_root',
      },
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: vi.fn(),
    };
    render(<NewRunParametersV2 {...props} />);

    screen.getByDisplayValue('gs://dummy_pipeline_root');
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: vi.fn(),
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
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: vi.fn(),
      setIsValidInput: vi.fn(),
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

  // Test for fix: Default parameters not displayed in Compare Runs
  // https://github.com/kubeflow/pipelines/issues/12536
  it('calls handleParameterChange with default values on mount', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        strParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'default string',
        },
        intParam: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 42,
        },
        boolParam: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
          defaultValue: true,
        },
        listParam: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
          defaultValue: [1, 2, 3],
        },
        structParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
          defaultValue: { key: 'value' },
        },
      },
      clonedRuntimeConfig: {},
      handlePipelineRootChange: vi.fn(),
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    // Verify that handleParameterChange was called on mount with all default values
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenCalledWith({
      strParam: 'default string',
      intParam: 42,
      boolParam: true,
      listParam: [1, 2, 3],
      structParam: { key: 'value' },
    });

    // Verify that the default values are displayed in the UI
    screen.getByDisplayValue('default string');
    screen.getByDisplayValue('42');
    screen.getByDisplayValue('true');
    screen.getByDisplayValue('[1,2,3]');
    screen.getByDisplayValue('{"key":"value"}');
  });
});

describe('Bug Fix: Default Parameters in Compare Runs (#12536)', () => {
  it('SCENARIO 1: User creates run with ALL default parameters (no changes)', () => {
    const handleParameterChangeSpy = vi.fn();

    const props = {
      titleMessage: 'Specify parameters required by the pipeline',
      specParameters: {
        string_param: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'default_string_value',
        },
        integer_param: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 42,
        },
        boolean_param: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
          defaultValue: true,
        },
        float_param: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_DOUBLE,
          defaultValue: 3.14,
        },
        list_param: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
          defaultValue: [1, 2, 3],
        },
        struct_param: {
          parameterType: ParameterType_ParameterTypeEnum.STRUCT,
          defaultValue: { key: 'value', nested: { data: 123 } },
        },
      },
      clonedRuntimeConfig: {},
      handleParameterChange: handleParameterChangeSpy,
    };

    // User does NOT interact - just renders form
    render(<NewRunParametersV2 {...props} />);

    // KEY ASSERTION: handleParameterChange called on mount (not 0!)
    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);

    // ALL default parameters sent to API
    expect(handleParameterChangeSpy).toHaveBeenCalledWith({
      string_param: 'default_string_value',
      integer_param: 42,
      boolean_param: true,
      float_param: 3.14,
      list_param: [1, 2, 3],
      struct_param: { key: 'value', nested: { data: 123 } },
    });
  });

  it('SCENARIO 2: User changes ONE parameter, others remain at default', () => {
    const handleParameterChangeSpy = vi.fn();

    const props = {
      titleMessage: 'Test',
      specParameters: {
        param_a: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'default_a',
        },
        param_b: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 100,
        },
        param_c: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
          defaultValue: false,
        },
      },
      clonedRuntimeConfig: {},
      handleParameterChange: handleParameterChangeSpy,
    };

    render(<NewRunParametersV2 {...props} />);

    // On mount, all defaults sent
    expect(handleParameterChangeSpy).toHaveBeenCalledWith({
      param_a: 'default_a',
      param_b: 100,
      param_c: false,
    });

    // User changes only param_a
    const paramAInput = screen.getByDisplayValue('default_a');
    handleParameterChangeSpy.mockClear();
    fireEvent.change(paramAInput, { target: { value: 'custom_value' } });

    // ALL parameters sent (not just changed one)
    expect(handleParameterChangeSpy).toHaveBeenCalledWith({
      param_a: 'custom_value',
      param_b: 100,
      param_c: false,
    });
  });

  it('SCENARIO 3: Falsy default values (0, false, empty list) are included', () => {
    const handleParameterChangeSpy = vi.fn();

    const props = {
      titleMessage: 'Test falsy defaults',
      specParameters: {
        zero_param: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 0,
        },
        false_param: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
          defaultValue: false,
        },
        zero_float: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_DOUBLE,
          defaultValue: 0.0,
        },
        empty_list: {
          parameterType: ParameterType_ParameterTypeEnum.LIST,
          defaultValue: [],
        },
      },
      clonedRuntimeConfig: {},
      handleParameterChange: handleParameterChangeSpy,
    };

    render(<NewRunParametersV2 {...props} />);

    // CRITICAL: Falsy values NOT omitted
    expect(handleParameterChangeSpy).toHaveBeenCalledWith({
      zero_param: 0,
      false_param: false,
      zero_float: 0.0,
      empty_list: [],
    });
  });
});

describe('Literal Parameter Dropdown (#12603)', () => {
  it('renders dropdown for string literal parameters with default value', () => {
    const props = {
      titleMessage: 'default Title',
      pipelineRoot: 'default pipelineRoot',
      specParameters: {
        literalParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'A',
          literals: ['A', 'B', 'C'],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
    };
    render(<NewRunParametersV2 {...props} />);

    // Label should be rendered
    const labels = screen.getAllByText('literalParam - string');
    expect(labels.length).toBeGreaterThan(0);

    // Default value is shown as selected
    screen.getByText('A');
    const input = screen.getByDisplayValue('A');
    expect(input).toBeInTheDocument();

    // Open the dropdown and verify all options are present
    const selectButton = screen.getByText('A');
    fireEvent.mouseDown(selectButton);
    screen.getByText('B');
    screen.getByText('C');
  });

  it('renders dropdown for integer literal parameters', () => {
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        replicaCount: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          defaultValue: 1,
          literals: [1, 3, 5],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
    };
    render(<NewRunParametersV2 {...props} />);

    // Default value is shown
    screen.getByText('1');

    // Open dropdown and verify all options
    const selectButton = screen.getByText('1');
    fireEvent.mouseDown(selectButton);
    screen.getByText('3');
    screen.getByText('5');
  });

  it('renders dropdown for float literal parameters', () => {
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        threshold: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_DOUBLE,
          defaultValue: 0.5,
          literals: [0.1, 0.5, 0.9],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
    };
    render(<NewRunParametersV2 {...props} />);

    screen.getByText('0.5');

    const selectButton = screen.getByText('0.5');
    fireEvent.mouseDown(selectButton);
    screen.getByText('0.1');
    screen.getByText('0.9');
  });

  it('does NOT render dropdown when literals array is empty', () => {
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        normalParam: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'hello',
          literals: [],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
    };
    const { container } = render(<NewRunParametersV2 {...props} />);

    // Should render a regular text field, not a select
    const textInput = screen.getByDisplayValue('hello');
    expect(textInput.tagName).toBe('INPUT');
    // No <select> element should exist
    expect(container.querySelector('[role="button"]')).toBeNull();
  });

  it('fires handleParameterChange when a dropdown value is selected', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        env: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'dev',
          literals: ['dev', 'staging', 'prod'],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
      handleParameterChange: handleParameterChangeSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    // On mount, default is propagated
    expect(handleParameterChangeSpy).toHaveBeenCalledWith({ env: 'dev' });

    // Open dropdown and select a different value
    const selectButton = screen.getByText('dev');
    fireEvent.mouseDown(selectButton);
    handleParameterChangeSpy.mockClear();
    fireEvent.click(screen.getByText('staging'));

    expect(handleParameterChangeSpy).toHaveBeenCalledTimes(1);
    expect(handleParameterChangeSpy).toHaveBeenCalledWith({ env: 'staging' });
  });

  it('marks input as invalid when literal parameter has no default value', () => {
    const setIsValidInputSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        env: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          literals: ['dev', 'staging', 'prod'],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
      handleParameterChange: vi.fn(),
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    // Without a default value, the form should be invalid (run button disabled)
    expect(setIsValidInputSpy).toHaveBeenCalledWith(false);
  });

  it('pre-selects the correct value from cloned RuntimeConfig for literal parameter', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        env: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          defaultValue: 'dev',
          literals: ['dev', 'staging', 'prod'],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: { parameters: { env: 'prod' } },
      handleParameterChange: handleParameterChangeSpy,
      setIsValidInput: vi.fn(),
    };
    render(<NewRunParametersV2 {...props} />);

    // The cloned value 'prod' should be displayed, not the default 'dev'
    screen.getByText('prod');
    screen.getByDisplayValue('prod');

    // handleParameterChange should be called with the cloned value
    expect(handleParameterChangeSpy).toHaveBeenCalledWith({ env: 'prod' });
  });

  it('marks input as valid after selecting a value from a no-default literal dropdown', () => {
    const setIsValidInputSpy = vi.fn();
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        env: {
          parameterType: ParameterType_ParameterTypeEnum.STRING,
          literals: ['dev', 'staging', 'prod'],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
      handleParameterChange: handleParameterChangeSpy,
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    // Initially invalid
    expect(setIsValidInputSpy).toHaveBeenCalledWith(false);

    // Open dropdown (use the label text since no value is selected yet)
    const labels = screen.getAllByText('env - string');
    // Find the select element trigger - the rendered display div
    const selectElements = document.querySelectorAll('[role="button"]');
    expect(selectElements.length).toBeGreaterThan(0);
    fireEvent.mouseDown(selectElements[0]);

    // Select a value
    setIsValidInputSpy.mockClear();
    fireEvent.click(screen.getByText('dev'));

    // Should now be valid
    expect(setIsValidInputSpy).toHaveBeenCalledWith(true);
  });

  it('renders dropdown for boolean literal parameters and validates selection', () => {
    const setIsValidInputSpy = vi.fn();
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        boolFlag: {
          parameterType: ParameterType_ParameterTypeEnum.BOOLEAN,
          literals: [true, false],
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
      handleParameterChange: handleParameterChangeSpy,
      setIsValidInput: setIsValidInputSpy,
    };
    render(<NewRunParametersV2 {...props} />);

    // Initially invalid (no default value)
    expect(setIsValidInputSpy).toHaveBeenCalledWith(false);

    // Open dropdown
    const selectElements = document.querySelectorAll('[role="button"]');
    expect(selectElements.length).toBeGreaterThan(0);
    fireEvent.mouseDown(selectElements[0]);

    // Select 'true'
    setIsValidInputSpy.mockClear();
    fireEvent.click(screen.getByText('true'));

    // Should now be valid
    expect(setIsValidInputSpy).toHaveBeenCalledWith(true);
  });

  it('does not show placeholder when literal default value is 0', () => {
    const handleParameterChangeSpy = vi.fn();
    const props = {
      titleMessage: 'default Title',
      specParameters: {
        count: {
          parameterType: ParameterType_ParameterTypeEnum.NUMBER_INTEGER,
          literals: [0, 1, 2],
          defaultValue: 0,
          isOptional: false,
          description: '',
        },
      },
      clonedRuntimeConfig: {},
      handleParameterChange: handleParameterChangeSpy,
      setIsValidInput: vi.fn(),
    };
    render(<NewRunParametersV2 {...props} />);

    // The placeholder "Select a value" should NOT appear when 0 is selected
    expect(screen.queryByText('Select a value')).toBeNull();

    // The selected value should be rendered as '0'
    expect(screen.getByText('0')).toBeInTheDocument();
  });
});
