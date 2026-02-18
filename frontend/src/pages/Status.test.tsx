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

import * as Utils from '../lib/Utils';
import { statusToIcon } from './Status';
import { NodePhase } from '../lib/StatusUtils';
import { render } from '@testing-library/react';
import { vi } from 'vitest';

describe('Status', () => {
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const startDate = new Date('Wed Jan 2 2019 9:10:11 GMT-0800');
  const endDate = new Date('Thu Jan 3 2019 10:11:12 GMT-0800');

  beforeEach(() => {
    vi.spyOn(Utils, 'formatDateString').mockImplementation((date: Date) => {
      return date === startDate ? '1/2/2019, 9:10:11 AM' : '1/3/2019, 10:11:12 AM';
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('statusToIcon', () => {
    it('handles an unknown phase', () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementationOnce(() => null);
      const { asFragment } = render(statusToIcon('bad phase' as any));
      expect(asFragment()).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
    });

    it('handles an undefined phase', () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementationOnce(() => null);
      const { asFragment } = render(statusToIcon(/* no phase */));
      expect(asFragment()).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', undefined);
    });

    it('handles ERROR phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.ERROR));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles FAILED phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.FAILED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles PENDING phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.PENDING));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles RUNNING phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.RUNNING));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles TERMINATING phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.TERMINATING));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles SKIPPED phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.SKIPPED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles SUCCEEDED phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.SUCCEEDED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles CACHED phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.CACHED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles TERMINATED phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.TERMINATED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles OMITTED phase', () => {
      const { asFragment } = render(statusToIcon(NodePhase.OMITTED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('displays start and end dates if both are provided', () => {
      const { asFragment } = render(statusToIcon(NodePhase.SUCCEEDED, startDate, endDate));
      expect(asFragment()).toMatchSnapshot();
    });

    it('does not display a end date if none was provided', () => {
      const { asFragment } = render(statusToIcon(NodePhase.SUCCEEDED, startDate));
      expect(asFragment()).toMatchSnapshot();
    });

    it('does not display a start date if none was provided', () => {
      const { asFragment } = render(statusToIcon(NodePhase.SUCCEEDED, undefined, endDate));
      expect(asFragment()).toMatchSnapshot();
    });

    it('does not display any dates if neither was provided', () => {
      const { asFragment } = render(statusToIcon(NodePhase.SUCCEEDED /* No dates */));
      expect(asFragment()).toMatchSnapshot();
    });

    Object.keys(NodePhase).forEach(status =>
      it('renders an icon with tooltip for phase: ' + status, () => {
        const { asFragment } = render(statusToIcon(NodePhase[status as keyof typeof NodePhase]));
        expect(asFragment()).toMatchSnapshot();
      }),
    );
  });
});
