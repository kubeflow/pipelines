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
import BusyButton from './BusyButton';
import TestIcon from '@material-ui/icons/Help';
import { create } from 'react-test-renderer';

describe('BusyButton', () => {
  it('renders with just a title', () => {
    const tree = create(<BusyButton title='test busy button' />);
    expect(tree).toMatchSnapshot();
  });

  it('renders with a title and icon', () => {
    const tree = create(<BusyButton title='test busy button' icon={TestIcon} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders with a title and icon, and busy', () => {
    const tree = create(<BusyButton title='test busy button' icon={TestIcon} busy={true} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders disabled', () => {
    const tree = create(<BusyButton title='test busy button' icon={TestIcon} disabled={true} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders a primary outlined buton', () => {
    const tree = create(<BusyButton title='test busy button' outlined={true} />);
    expect(tree).toMatchSnapshot();
  });
});
