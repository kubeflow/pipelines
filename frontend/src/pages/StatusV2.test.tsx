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

describe('Status', () => {
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

  const startDate = new Date('Wed Jan 2 2019 9:10:11 GMT-0800');
  const endDate = new Date('Thu Jan 3 2019 10:11:12 GMT-0800');

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date: string | Date | undefined) => {
      return date === startDate ? '1/2/2019, 9:10:11 AM' : '1/3/2019, 10:11:12 AM';
    });
  });

  describe('statusToIcon', () => {
    it('handles an unknown state', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      const { asFragment } = render(statusToIcon('bad state' as any));
      expect(asFragment()).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown state:', 'bad state');
    });

    it('handles an undefined state', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      const { asFragment } = render(statusToIcon(/* no phase */));
      expect(asFragment()).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown state:', undefined);
    });

    // TODO: Enable this test after react-scripts is upgraded to v4.0.0
    // it('react testing for ERROR state', async () => {
    //   const { findByText, getByTestId } = render(
    //     statusToIcon(NodePhase.ERROR),
    //   );

    //   fireEvent.mouseOver(getByTestId('node-status-sign'));
    //   findByText('Error while running this resource');
    // });

    it('handles FAILED state', () => {
      const { container } = render(statusToIcon(V2beta1RuntimeState.FAILED));
      expect(container.firstChild).toMatchInlineSnapshot(`
        <div
          class=""
          data-mui-internal-clone-element="true"
        >
          <svg
            aria-hidden="true"
            class="MuiSvgIcon-root MuiSvgIcon-fontSizeMedium css-i4bv87-MuiSvgIcon-root"
            data-testid="node-status-sign"
            focusable="false"
            style="color: rgb(213, 0, 0); height: 18px; width: 18px;"
            viewBox="0 0 24 24"
          >
            <path
              d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2m1 15h-2v-2h2zm0-4h-2V7h2z"
            />
          </svg>
        </div>
      `);
    });

    it('handles PENDING state', () => {
      const { container } = render(statusToIcon(V2beta1RuntimeState.PENDING));
      expect(container.firstChild).toMatchInlineSnapshot(`
        <div
          class=""
          data-mui-internal-clone-element="true"
        >
          <svg
            aria-hidden="true"
            class="MuiSvgIcon-root MuiSvgIcon-fontSizeMedium css-i4bv87-MuiSvgIcon-root"
            data-testid="node-status-sign"
            focusable="false"
            style="color: rgb(154, 160, 166); height: 18px; width: 18px;"
            viewBox="0 0 24 24"
          >
            <path
              d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2M12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8"
            />
            <path
              d="M12.5 7H11v6l5.25 3.15.75-1.23-4.5-2.67z"
            />
          </svg>
        </div>
      `);
    });

    it('handles RUNNING state', () => {
      const { container } = render(statusToIcon(V2beta1RuntimeState.RUNNING));
      expect(container.firstChild).toMatchSnapshot();
    });

    it('handles CANCELING state', () => {
      const { container } = render(statusToIcon(V2beta1RuntimeState.CANCELING));
      expect(container.firstChild).toMatchSnapshot();
    });

    it('handles SKIPPED state', () => {
      const { container } = render(statusToIcon(V2beta1RuntimeState.SKIPPED));
      expect(container.firstChild).toMatchInlineSnapshot(`
        <div
          class=""
          data-mui-internal-clone-element="true"
        >
          <svg
            aria-hidden="true"
            class="MuiSvgIcon-root MuiSvgIcon-fontSizeMedium css-i4bv87-MuiSvgIcon-root"
            data-testid="node-status-sign"
            focusable="false"
            style="color: rgb(95, 99, 104); height: 18px; width: 18px;"
            viewBox="0 0 24 24"
          >
            <path
              d="m6 18 8.5-6L6 6zM16 6v12h2V6z"
            />
          </svg>
        </div>
      `);
    });

    it('handles SUCCEEDED state', () => {
      const { container } = render(statusToIcon(V2beta1RuntimeState.SUCCEEDED));
      expect(container.firstChild).toMatchInlineSnapshot(`
        <div
          class=""
          data-mui-internal-clone-element="true"
        >
          <svg
            aria-hidden="true"
            class="MuiSvgIcon-root MuiSvgIcon-fontSizeMedium css-i4bv87-MuiSvgIcon-root"
            data-testid="node-status-sign"
            focusable="false"
            style="color: rgb(52, 168, 83); height: 18px; width: 18px;"
            viewBox="0 0 24 24"
          >
            <path
              d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2m-2 15-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8z"
            />
          </svg>
        </div>
      `);
    });

    it('handles CANCELED state', () => {
      const { container } = render(statusToIcon(V2beta1RuntimeState.CANCELED));
      expect(container.firstChild).toMatchSnapshot();
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

    Object.keys(V2beta1RuntimeState).map(status =>
      it('renders an icon with tooltip for phase: ' + status, () => {
        const { asFragment } = render(
          statusToIcon(V2beta1RuntimeState[status as keyof typeof V2beta1RuntimeState]),
        );
        expect(asFragment()).toMatchSnapshot();
      }),
    );
  });
});
