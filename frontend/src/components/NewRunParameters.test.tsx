/*
 * Copyright 2019 The Kubeflow Authors
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

import * as React from 'react';
import { mount, shallow } from 'enzyme';
import NewRunParameters, { NewRunParametersProps } from './NewRunParameters';

describe('NewRunParameters', () => {
  it('shows parameters', () => {
    const props = {
      handleParamChange: jest.fn(),
      initialParams: [{ name: 'testParam', value: 'testVal' }],
      titleMessage: 'Specify parameters required by the pipeline',
    } as NewRunParametersProps;
    expect(shallow(<NewRunParameters {...props} />)).toMatchSnapshot();
  });

  it('does not display any text fields if there are no parameters', () => {
    const props = {
      handleParamChange: jest.fn(),
      initialParams: [],
      titleMessage: 'This pipeline has no parameters',
    } as NewRunParametersProps;
    expect(shallow(<NewRunParameters {...props} />)).toMatchSnapshot();
  });

  it('clicking the open editor button for json parameters displays an editor', () => {
    const handleParamChange = jest.fn();
    const props = {
      handleParamChange,
      initialParams: [{ name: 'testParam', value: '{"test":"value"}' }],
      titleMessage: 'Specify json parameters required by the pipeline',
    } as NewRunParametersProps;
    const tree = mount(<NewRunParameters {...props} />);
    tree
      .findWhere(el => el.text() === 'Open Json Editor')
      .hostNodes()
      .find('Button')
      .simulate('click');
    expect(handleParamChange).toHaveBeenCalledTimes(1);
    expect(handleParamChange).toHaveBeenLastCalledWith(0, '{\n  "test": "value"\n}');
    expect(tree.find('Editor')).toMatchSnapshot();
  });

  it('fires handleParamChange callback on change', () => {
    const handleParamChange = jest.fn();
    const props = {
      handleParamChange,
      initialParams: [
        { name: 'testParam1', value: 'testVal1' },
        { name: 'testParam2', value: 'testVal2' },
      ],
      titleMessage: 'Specify parameters required by the pipeline',
    } as NewRunParametersProps;

    const tree = mount(<NewRunParameters {...props} />);
    tree
      .find('input#newRunPipelineParam1')
      .simulate('change', { target: { value: 'test param value' } });
    expect(handleParamChange).toHaveBeenCalledTimes(1);
    expect(handleParamChange).toHaveBeenLastCalledWith(1, 'test param value');
  });
});
