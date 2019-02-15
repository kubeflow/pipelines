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
import { NodePhase, hasFinished, statusToFadedColor, statusToIcon } from './Status';
import { color } from '../Css';
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
    [NodePhase.ERROR, NodePhase.FAILED, NodePhase.SUCCEEDED, NodePhase.SKIPPED].forEach(status => {
      it(`returns \'true\' if status is: ${status}`, () => {
        expect(hasFinished(status)).toBe(true);
      });
    });

    [NodePhase.PENDING, NodePhase.RUNNING, NodePhase.UNKNOWN].forEach(status => {
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

  describe('statusToFadedColor', () => {
    it('handles an invalid phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToFadedColor('bad phase' as any)).toEqual(color.fadedStatusColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'bad phase');
    });

    it('handles an \'Unknown\' phase', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementationOnce(() => null);
      expect(statusToFadedColor(NodePhase.UNKNOWN)).toEqual(color.fadedStatusColors.notStarted);
      expect(consoleSpy).toHaveBeenLastCalledWith('Unknown node phase:', 'Unknown');
    });

    it('returns color \'not started\' if status is undefined', () => {
      expect(statusToFadedColor(undefined)).toEqual(color.fadedStatusColors.notStarted);
    });

    it('returns color \'not started\' if status is \'Pending\'', () => {
      expect(statusToFadedColor(NodePhase.PENDING)).toEqual(color.fadedStatusColors.notStarted);
    });

    [NodePhase.ERROR, NodePhase.FAILED].forEach(status => {
      it(`returns color \'error\' if status is: ${status}`, () => {
        expect(statusToFadedColor(status)).toEqual(color.fadedStatusColors.error);
      });
    });

    it('returns color \'running\' if status is \'Running\'', () => {
      expect(statusToFadedColor(NodePhase.RUNNING)).toEqual(color.fadedStatusColors.running);
    });

    it('returns color \'stop or skip\' if status is \'Skipped\'', () => {
      expect(statusToFadedColor(NodePhase.SKIPPED)).toEqual(color.fadedStatusColors.stopOrSkip);
    });

    it('returns color \'succeeded\' if status is \'Succeeded\'', () => {
      expect(statusToFadedColor(NodePhase.SUCCEEDED)).toEqual(color.fadedStatusColors.succeeded);
    });
  });
});
