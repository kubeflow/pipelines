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

import DetailsTable from './DetailsTable';
import { shallow } from 'enzyme';

describe('DetailsTable', () => {
  it('shows no rows', () => {
    const tree = shallow(<DetailsTable fields={[]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows one row', () => {
    const tree = shallow(<DetailsTable fields={[['key', 'value']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows a row with a title', () => {
    const tree = shallow(<DetailsTable title='some title' fields={[['key', 'value']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows key and value for large values', () => {
    const tree = shallow(
      <DetailsTable
        fields={[
          [
            'key',
            `Lorem Ipsum is simply dummy text of the printing and typesetting
      industry. Lorem Ipsum has been the industry's standard dummy text ever
      since the 1500s, when an unknown printer took a galley of type and
      scrambled it to make a type specimen book. It has survived not only five
      centuries, but also the leap into electronic typesetting, remaining
      essentially unchanged. It was popularised in the 1960s with the release
      of Letraset sheets containing Lorem Ipsum passages, and more recently
      with desktop publishing software like Aldus PageMaker including versions
      of Lorem Ipsum.`,
          ],
        ]}
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('shows key and value in row', () => {
    const tree = shallow(<DetailsTable fields={[['key', 'value']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows key and JSON value in row', () => {
    const tree = shallow(
      <DetailsTable fields={[['key', JSON.stringify([{ jsonKey: 'jsonValue' }])]]} />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('does render arrays as JSON', () => {
    const tree = shallow(<DetailsTable fields={[['key', '[]']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does render arrays as JSON', () => {
    const tree = shallow(<DetailsTable fields={[['key', '{}']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not render nulls as JSON', () => {
    const tree = shallow(<DetailsTable fields={[['key', 'null']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not render numbers as JSON', () => {
    const tree = shallow(<DetailsTable fields={[['key', '10']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not render strings as JSON', () => {
    const tree = shallow(<DetailsTable fields={[['key', '"some string"']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('does not render booleans as JSON', () => {
    const tree = shallow(<DetailsTable fields={[['key1', 'true'], ['key2', 'false']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows keys and values for multiple rows', () => {
    const tree = shallow(
      <DetailsTable
        fields={[
          ['key1', 'value1'],
          ['key2', JSON.stringify([{ jsonKey: 'jsonValue2' }])],
          ['key3', 'value3'],
          ['key4', 'value4'],
          ['key5', JSON.stringify({ jsonKey: { nestedJsonKey: 'jsonValue' } })],
          ['key6', 'value6'],
          ['key6', 'value7'],
        ]}
      />,
    );
    expect(tree).toMatchSnapshot();
  });

  it('does render values with the provided valueComponent', () => {
    const valueComponent: React.FC<any> = ({ key }) => <a>{key}</a>;
    const tree = shallow(
      <DetailsTable fields={[['key', { key: 'foobar' } as any]]} valueComponent={valueComponent} />,
    );
    expect(tree).toMatchSnapshot();
  });
});
