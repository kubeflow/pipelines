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
import { shallow } from 'enzyme';

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
    it('handles an unknown phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      const tree = shallow(statusToIcon('bad phase' as any));
      expect(tree).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
    });

    it('handles an undefined phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      const tree = shallow(statusToIcon(/* no phase */));
      expect(tree).toMatchSnapshot();
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', undefined);
    });

    // TODO: Enable this test after react-scripts is upgraded to v4.0.0
    // it('react testing for ERROR phase', async () => {
    //   const { findByText, getByTestId } = render(
    //     statusToIcon(NodePhase.ERROR),
    //   );

    //   fireEvent.mouseOver(getByTestId('node-status-sign'));
    //   findByText('Error while running this resource');
    // });

    it('handles ERROR phase', () => {
      const tree = shallow(statusToIcon(NodePhase.ERROR));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
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
        </span>
      `);
    });

    it('handles FAILED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.FAILED));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
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
        </span>
      `);
    });

    it('handles PENDING phase', () => {
      const tree = shallow(statusToIcon(NodePhase.PENDING));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
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
        </span>
      `);
    });

    it('handles RUNNING phase', () => {
      const tree = shallow(statusToIcon(NodePhase.RUNNING));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
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
        </span>
      `);
    });

    it('handles TERMINATING phase', () => {
      const tree = shallow(statusToIcon(NodePhase.TERMINATING));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
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
        </span>
      `);
    });

    it('handles SKIPPED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.SKIPPED));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
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
        </span>
      `);
    });

    it('handles SUCCEEDED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
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
        </span>
      `);
    });

    it('handles CACHED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.CACHED));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
          <StatusCached
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#34a853",
                "height": 18,
                "width": 18,
              }
            }
          />
        </span>
      `);
    });

    it('handles TERMINATED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.TERMINATED));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
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
        </span>
      `);
    });

    it('handles OMITTED phase', () => {
      const tree = shallow(statusToIcon(NodePhase.OMITTED));
      expect(tree.find('span')).toMatchInlineSnapshot(`
        <span
          style={
            Object {
              "height": 18,
            }
          }
        >
          <pure(BlockIcon)
            data-testid="node-status-sign"
            style={
              Object {
                "color": "#5f6368",
                "height": 18,
                "width": 18,
              }
            }
          />
        </span>
      `);
    });

    it('displays start and end dates if both are provided', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, startDate, endDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display a end date if none was provided', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, startDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display a start date if none was provided', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED, undefined, endDate));
      expect(tree).toMatchSnapshot();
    });

    it('does not display any dates if neither was provided', () => {
      const tree = shallow(statusToIcon(NodePhase.SUCCEEDED /* No dates */));
      expect(tree).toMatchSnapshot();
    });

    Object.keys(NodePhase).map(status =>
      it('renders an icon with tooltip for phase: ' + status, () => {
        const tree = shallow(statusToIcon(NodePhase[status]));
        expect(tree).toMatchSnapshot();
      }),
    );
  });
});
