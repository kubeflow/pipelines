/*
 * Copyright 2018 The Kubeflow Authors
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

import BusyButton from './BusyButton';
import TestIcon from '@mui/icons-material/Help';
import { render } from '@testing-library/react';

describe('BusyButton', () => {
  it('renders with just a title', () => {
    const { asFragment } = render(<BusyButton title='test busy button' />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders with a title and icon', () => {
    const { asFragment } = render(<BusyButton title='test busy button' icon={TestIcon} />);
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders with a title and icon, and busy', () => {
    const { asFragment } = render(
      <BusyButton title='test busy button' icon={TestIcon} busy={true} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders disabled', () => {
    const { asFragment } = render(
      <BusyButton title='test busy button' icon={TestIcon} disabled={true} />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders a primary outlined buton', () => {
    const { asFragment } = render(<BusyButton title='test busy button' outlined={true} />);
    expect(asFragment()).toMatchSnapshot();
  });
});
