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

import * as Utils from '../lib/Utils';
import { NodePhase, hasFinished, statusBgColors, statusToBgColor, statusToIcon, checkIfTerminated } from './Status';
import { shallow } from 'enzyme';


describe('Status', () => {
  // We mock this because it uses toLocaleDateString, which causes mismatches between local and CI
  // test enviroments
  const formatDateStringSpy = jest.spyOn(Utils, 'formatDateString');

  const startDate = new Date('Wed Jan 2 2019 9:10:11 GMT-0800');
  const endDate = new Date('Thu Jan 3 2019 10:11:12 GMT-0800');

  beforeEach(() => {
    formatDateStringSpy.mockImplementation((date: Date) => {
      return (date === startDate) ? '1/2/2019, 9:10:11 AM' : '1/3/2019, 10:11:12 AM';
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

    Object.keys(NodePhase).map(status => (
      it('renders an icon with tooltip for phase: ' + status, () => {
        const tree = shallow(statusToIcon(NodePhase[status]));
        expect(tree).toMatchSnapshot();
      })
    ));
  });

  describe('hasFinished', () => {
    [NodePhase.ERROR, NodePhase.FAILED, NodePhase.SUCCEEDED, NodePhase.SKIPPED, NodePhase.TERMINATED].forEach(status => {
      it(`returns \'true\' if status is: ${status}`, () => {
        expect(hasFinished(status)).toBe(true);
      });
    });

    [NodePhase.PENDING, NodePhase.RUNNING, NodePhase.UNKNOWN, NodePhase.TERMINATING].forEach(status => {
      it(`returns \'false\' if status is: ${status}`, () => {
        expect(hasFinished(status)).toBe(false);
      });
    });

    it('returns \'false\' if status is undefined', () => {
      expect(hasFinished(undefined)).toBe(false);
    });

    it('returns \'false\' if status is invalid', () => {
      expect(hasFinished('bad phase' as any)).toBe(false);
    });
  });

  describe('statusToBgColor', () => {
    it('handles an invalid phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor('bad phase' as any)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
    });

    it('handles an \'Unknown\' phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor(NodePhase.UNKNOWN)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'Unknown');
    });

    it('returns color \'not started\' if status is undefined', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToBgColor(undefined)).toEqual(statusBgColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', undefined);
    });

    it('returns color \'not started\' if status is \'Pending\'', () => {
      expect(statusToBgColor(NodePhase.PENDING)).toEqual(statusBgColors.notStarted);
    });

    [NodePhase.ERROR, NodePhase.FAILED].forEach(status => {
      it(`returns color \'error\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.error);
      });
    });

    [NodePhase.RUNNING, NodePhase.TERMINATING].forEach(status => {
      it(`returns color \'running\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.running);
      });
    });

    [NodePhase.SKIPPED, NodePhase.TERMINATED].forEach(status => {
      it(`returns color \'terminated or skipped\' if status is: ${status}`, () => {
        expect(statusToBgColor(status)).toEqual(statusBgColors.terminatedOrSkipped);
      });
    });

    it('returns color \'succeeded\' if status is \'Succeeded\'', () => {
      expect(statusToBgColor(NodePhase.SUCCEEDED)).toEqual(statusBgColors.succeeded);
    });
  });

  describe('checkIfTerminated', () => {
    it('returns status \'terminated\' if status is \'failed\' and error message is \'terminated\'', () => {
      expect(checkIfTerminated(NodePhase.FAILED, 'terminated')).toEqual(NodePhase.TERMINATED);
    });

    [
      NodePhase.SUCCEEDED,
      NodePhase.ERROR,
      NodePhase.SKIPPED,
      NodePhase.PENDING,
      NodePhase.RUNNING,
      NodePhase.TERMINATING,
      NodePhase.UNKNOWN
    ].forEach(status => {
      it(`returns the original status, even if message is 'terminated', if status is: ${status}`, () => {
        expect(checkIfTerminated(status, 'terminated')).toEqual(status);
      });
    });

    it('returns \'failed\' if status is \'failed\' and no error message is provided', () => {
      expect(checkIfTerminated(NodePhase.FAILED)).toEqual(NodePhase.FAILED);
    });

    it('returns \'failed\' if status is \'failed\' and empty error message is provided', () => {
      expect(checkIfTerminated(NodePhase.FAILED, '')).toEqual(NodePhase.FAILED);
    });

    it('returns \'failed\' if status is \'failed\' and arbitrary error message is provided', () => {
      expect(checkIfTerminated(NodePhase.FAILED, 'some random error')).toEqual(NodePhase.FAILED);
    });
  });
});
