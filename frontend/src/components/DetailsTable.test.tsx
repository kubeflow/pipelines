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
import { create } from 'react-test-renderer';

describe('DetailsTable', () => {
  it('shows no rows', () => {
    const tree = create(<DetailsTable fields={[]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows one row', () => {
    const tree = create(<DetailsTable fields={[['key', 'value']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows a row with a title', () => {
    const tree = create(<DetailsTable title='some title' fields={[['key', 'value']]} />);
    expect(tree).toMatchSnapshot();
  });

  it('shows key and value for large values', () => {
    const tree = create(<DetailsTable fields={[
      ['key', `Lorem Ipsum is simply dummy text of the printing and typesetting
      industry. Lorem Ipsum has been the industry's standard dummy text ever
      since the 1500s, when an unknown printer took a galley of type and
      scrambled it to make a type specimen book. It has survived not only five
      centuries, but also the leap into electronic typesetting, remaining
      essentially unchanged. It was popularised in the 1960s with the release
      of Letraset sheets containing Lorem Ipsum passages, and more recently
      with desktop publishing software like Aldus PageMaker including versions
      of Lorem Ipsum.`]
    ]} />).toJSON();
    expect(tree).toMatchSnapshot();
  });

  it('shows key and value in row', () => {
    const tree = create(<DetailsTable fields={[['key', 'value']]} />).toJSON();
    expect(tree).toMatchSnapshot();
  });

  it('shows keys and values in a bunch of rows', () => {
    const tree = create(<DetailsTable fields={[
      ['key1', 'value1'],
      ['key2', 'value2'],
      ['key3', 'value3'],
      ['key4', 'value4'],
      ['key5', 'value5'],
      ['key6', 'value6'],
      ['key6', 'value7'],
    ]} />).toJSON();
    expect(tree).toMatchSnapshot();
  });
});
