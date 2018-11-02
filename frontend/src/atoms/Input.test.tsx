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

import * as React from 'react';

import Input from './Input';
import { shallow } from 'enzyme';
import toJson from 'enzyme-to-json';

const mockInstance = { state: {} } as any;

describe('Input', () => {
  it('renders with the right styles by default', () => {
    const tree = shallow(<Input instance={mockInstance} field='fieldname' />);
    expect(toJson(tree)).toMatchSnapshot();
  });

  it('accepts height and width as prop overrides', () => {
    const tree = shallow(<Input instance={mockInstance} height={123} width={456} field='fieldname' />);
    expect(toJson(tree)).toMatchSnapshot();
  });

  it('gets its value from the instance state field', () => {
    mockInstance.state.fieldname = 'field value';
    const tree = shallow(<Input instance={mockInstance} field='fieldname' />);
    expect(toJson(tree)).toMatchSnapshot();
  });

  it('calls the instance handleChange function if defined', () => {
    mockInstance.handleChange = jest.fn();
    const tree = shallow(<Input instance={mockInstance} field='fieldname' />);
    tree.simulate('change');
    expect(mockInstance.handleChange).toHaveBeenCalledWith('fieldname');
  });
});
