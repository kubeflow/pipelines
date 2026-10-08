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

import { render, screen } from '@testing-library/react';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { RunStatus } from './RunStatus';
import { RecentRunHealth } from './RecentRunHealth';

const expected: Record<V2beta1RuntimeState, string> = {
  SUCCEEDED: 'succeeded',
  RUNNING: 'running',
  FAILED: 'failed',
  PENDING: 'neutral',
  CANCELING: 'running',
  CANCELED: 'neutral',
  PAUSED: 'warning',
  SKIPPED: 'neutral',
  RUNTIME_STATE_UNSPECIFIED: 'neutral',
};

it.each(Object.values(V2beta1RuntimeState))(
  'uses the same %s tone in badges and recent-run health',
  (state) => {
    const { container } = render(
      <>
        <RunStatus state={state} />
        <RecentRunHealth runs={[{ state }]} />
      </>,
    );
    expect(container.querySelector('.kfp-run-status')).toHaveAttribute(
      'data-tone',
      expected[state],
    );
    expect(screen.getByRole('img').firstElementChild).toHaveAttribute('data-tone', expected[state]);
  },
);
