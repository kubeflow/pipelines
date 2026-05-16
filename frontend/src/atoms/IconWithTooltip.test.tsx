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

import IconWithTooltip from './IconWithTooltip';
import TestIcon from '@mui/icons-material/Help';
import { render } from '@testing-library/react';

describe('IconWithTooltip', () => {
  it('renders without height or weight', () => {
    const { asFragment } = render(
      <IconWithTooltip Icon={TestIcon} iconColor='green' tooltip='test icon tooltip' />,
    );
    expect(asFragment()).toMatchSnapshot();
  });

  it('renders with height and weight', () => {
    const { asFragment } = render(
      <IconWithTooltip
        Icon={TestIcon}
        height={20}
        width={20}
        iconColor='green'
        tooltip='test icon tooltip'
      />,
    );
    expect(asFragment()).toMatchSnapshot();
  });
});
