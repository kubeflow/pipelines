/*
 * Copyright 2023 The Kubeflow Authors
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

import * as Utils from 'src/lib/Utils';
import { statusToIcon } from './StatusV2';
import { render } from '@testing-library/react';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';
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
    it('handles an unknown state', () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementationOnce(() => null);
      const { asFragment } = render(statusToIcon('bad state' as any));
      expect(asFragment()).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown state:', 'bad state');
    });

    it('handles an undefined state', () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementationOnce(() => null);
      const { asFragment } = render(statusToIcon(/* no phase */));
      expect(asFragment()).toMatchSnapshot();
      expect(consoleSpy).not.toHaveBeenCalled();
    });

    it('handles FAILED state', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.FAILED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles PENDING state', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.PENDING));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles RUNNING state', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.RUNNING));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles CANCELING state', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.CANCELING));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles SKIPPED state', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.SKIPPED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles SUCCEEDED state', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.SUCCEEDED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('handles CANCELED state', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.CANCELED));
      expect(asFragment()).toMatchSnapshot();
    });

    it('displays start and end dates if both are provided', () => {
      const { asFragment } = render(
        statusToIcon(V2beta1RuntimeState.SUCCEEDED, startDate, endDate),
      );
      expect(asFragment()).toMatchSnapshot();
    });

    it('does not display a end date if none was provided', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.SUCCEEDED, startDate));
      expect(asFragment()).toMatchSnapshot();
    });

    it('does not display a start date if none was provided', () => {
      const { asFragment } = render(
        statusToIcon(V2beta1RuntimeState.SUCCEEDED, undefined, endDate),
      );
      expect(asFragment()).toMatchSnapshot();
    });

    it('does not display any dates if neither was provided', () => {
      const { asFragment } = render(statusToIcon(V2beta1RuntimeState.SUCCEEDED /* No dates */));
      expect(asFragment()).toMatchSnapshot();
    });

    Object.keys(V2beta1RuntimeState).forEach(status =>
      it('renders an icon with tooltip for phase: ' + status, () => {
        const { asFragment } = render(
          statusToIcon(V2beta1RuntimeState[status as keyof typeof V2beta1RuntimeState]),
        );
        expect(asFragment()).toMatchSnapshot();
      }),
    );
  });
});
