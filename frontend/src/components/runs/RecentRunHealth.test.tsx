/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { cleanup, render, screen } from '@testing-library/react';
import { V2beta1RuntimeState as State } from 'src/apisv2beta1/run';
import { RecentRunHealth } from './RecentRunHealth';

afterEach(cleanup);
it('labels the actual sample size and every returned state', () => {
  render(
    <RecentRunHealth
      runs={[
        { run_id: 'one', state: State.SUCCEEDED },
        { run_id: 'two', state: State.FAILED },
        { run_id: 'three', state: State.RUNNING },
      ]}
    />,
  );
  expect(
    screen.getByRole('img', { name: 'Last 3 runs: SUCCEEDED, FAILED, RUNNING' }),
  ).toBeVisible();
  expect(screen.getByText('3 recent runs')).toBeVisible();
  expect(screen.queryByText('5 runs')).not.toBeInTheDocument();
});
it('shows an honest empty sample rather than a success rate', () => {
  render(<RecentRunHealth runs={[]} />);
  expect(screen.getByText('No recent runs')).toBeVisible();
  expect(screen.queryByRole('img')).not.toBeInTheDocument();
});
