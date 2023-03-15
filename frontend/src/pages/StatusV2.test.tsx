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
import { shallow } from 'enzyme';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';

describe('Status', () => {
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

  const startDate = new Date('Wed Jan 2 2019 9:10:11 GMT-0800');
  const endDate = new Date('Thu Jan 3 2019 10:11:12 GMT-0800');

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date: Date) => {
      return date === startDate ? '1/2/2019, 9:10:11 AM' : '1/3/2019, 10:11:12 AM';
    });
  });

  describe('statusToIcon', () => {
    it('handles an unknown state', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      const tree = shallow(statusToIcon('bad state' as any));
      expect(tree).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown state:', 'bad state');
    });

    it('handles an undefined state', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      const tree = shallow(statusToIcon(/* no phase */));
      expect(tree).toMatchSnapshot();
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
      const tree = shallow(statusToIcon(V2beta1RuntimeState.FAILED));
      expect(tree.find('div')).toMatchInlineSnapshot(`
        <div>
          <pure(ErrorIcon)
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#d50000",
                "height": 18,
                "width": 18,
              }
            }
          />
        </div>
      `);
    });

    it('handles PENDING state', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.PENDING));
      expect(tree.find('div')).toMatchInlineSnapshot(`
        <div>
          <pure(ScheduleIcon)
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#9aa0a6",
                "height": 18,
                "width": 18,
              }
            }
          />
        </div>
      `);
    });

    it('handles RUNNING state', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.RUNNING));
      expect(tree.find('div')).toMatchInlineSnapshot(`
        <div>
          <StatusRunning
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#4285f4",
                "height": 18,
                "width": 18,
              }
            }
          />
        </div>
      `);
    });

    it('handles CANCELING state', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.CANCELING));
      expect(tree.find('div')).toMatchInlineSnapshot(`
        <div>
          <StatusRunning
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#4285f4",
                "height": 18,
                "width": 18,
              }
            }
          />
        </div>
      `);
    });

    it('handles SKIPPED state', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.SKIPPED));
      expect(tree.find('div')).toMatchInlineSnapshot(`
        <div>
          <pure(SkipNextIcon)
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#5f6368",
                "height": 18,
                "width": 18,
              }
            }
          />
        </div>
      `);
    });

    it('handles SUCCEEDED state', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.SUCCEEDED));
      expect(tree.find('div')).toMatchInlineSnapshot(`
        <div>
          <pure(CheckCircleIcon)
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#34a853",
                "height": 18,
                "width": 18,
              }
            }
          />
        </div>
      `);
    });

    it('handles CANCELED state', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.CANCELED));
      expect(tree.find('div')).toMatchInlineSnapshot(`
        <div>
          <StatusRunning
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#80868b",
                "height": 18,
                "width": 18,
              }
            }
          />
        </div>
      `);
    });

    it('displays start and end dates if both are provided', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.SUCCEEDED, startDate, endDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display a end date if none was provided', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.SUCCEEDED, startDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display a start date if none was provided', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.SUCCEEDED, undefined, endDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display any dates if neither was provided', () => {
      const tree = shallow(statusToIcon(V2beta1RuntimeState.SUCCEEDED /* No dates */));
      expect(tree).toMatchSnapshot();
    });

    Object.keys(V2beta1RuntimeState).map(status =>
      it('renders an icon with tooltip for phase: ' + status, () => {
        const tree = shallow(statusToIcon(V2beta1RuntimeState[status]));
        expect(tree).toMatchSnapshot();
      }),
    );
  });
});
