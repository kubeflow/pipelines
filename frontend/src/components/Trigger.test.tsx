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

import * as React from 'react';
import Trigger from './Trigger';
import { shallow } from 'enzyme';
import { TriggerType, PeriodicInterval } from '../lib/TriggerUtils';

describe('Trigger', () => {
  // tslint:disable-next-line:variable-name
  const RealDate = Date;

  function mockDate(isoDate: any): void {
    (global as any).Date = class extends RealDate {
      constructor() {
        super();
        return new RealDate(isoDate);
      }
    };
  }
  const testDate = new Date(2018, 11, 21, 7, 53);
  mockDate(testDate);

  it('renders periodic schedule controls for initial render', () => {
    const tree = shallow(<Trigger />);
    expect(tree).toMatchSnapshot();
  });

  it('renders periodic schedule controls if the trigger type is CRON', () => {
    const tree = shallow(<Trigger />);
    (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
    expect(tree).toMatchSnapshot();
  });

  it('renders week days if the trigger type is CRON and interval is weekly', () => {
    const tree = shallow(<Trigger />);
    (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
    (tree.instance() as Trigger).handleChange('intervalCategory')({ target: { value: PeriodicInterval.WEEK } });
    expect(tree).toMatchSnapshot();
  });

  it('renders all week days enabled', () => {
    const tree = shallow(<Trigger />);
    (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
    (tree.instance() as Trigger).handleChange('intervalCategory')({ target: { value: PeriodicInterval.WEEK } });
    (tree.instance() as any)._toggleCheckAllDays();
    expect(tree).toMatchSnapshot();
  });

  it('enables a single day on click', () => {
    const tree = shallow(<Trigger />);
    (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
    (tree.instance() as Trigger).handleChange('intervalCategory')({ target: { value: PeriodicInterval.WEEK } });
    (tree.instance() as any)._toggleDay(1);
    (tree.instance() as any)._toggleDay(3);
    expect(tree).toMatchSnapshot();
  });

  describe('interval trigger', () => {
    it('builds an every-minute trigger by default', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      expect(spy).toHaveBeenLastCalledWith(
        { periodic_schedule: { end_time: undefined, interval_second: '60', start_time: undefined } },
        '10'
      );
    });

    it('builds trigger with a start time if the checkbox is checked', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({ target: { type: 'checkbox', checked: true } });
      expect(spy).toHaveBeenLastCalledWith(
        { periodic_schedule: { end_time: undefined, interval_second: '60', start_time: testDate } },
        '10'
      );
    });

    it('builds trigger with the entered start date/time', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({ target: { type: 'checkbox', checked: true } });
      (tree.instance() as Trigger).handleChange('startDate')({ target: { value: '2018-11-23' } });
      (tree.instance() as Trigger).handleChange('endTime')({ target: { value: '08:35' } });
      expect(spy).toHaveBeenLastCalledWith(
        {
          periodic_schedule: {
            end_time: undefined,
            interval_second: '60',
            start_time: new Date(2018, 10, 23, 8, 35)
          }
        },
        '10'
      );
    });

    it('builds trigger without the entered start date if no time is entered', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({ target: { type: 'checkbox', checked: true } });
      (tree.instance() as Trigger).handleChange('startDate')({ target: { value: '2018-11-23' } });
      (tree.instance() as Trigger).handleChange('startTime')({ target: { value: '' } });
      expect(spy).toHaveBeenLastCalledWith(
        {
          periodic_schedule: {
            end_time: undefined,
            interval_second: '60',
            start_time: undefined,
          }
        },
        '10'
      );
    });

    it('builds trigger without the entered start time if no date is entered', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({ target: { type: 'checkbox', checked: true } });
      (tree.instance() as Trigger).handleChange('startDate')({ target: { value: '' } });
      (tree.instance() as Trigger).handleChange('startTime')({ target: { value: '11:33' } });
      expect(spy).toHaveBeenLastCalledWith(
        {
          periodic_schedule: {
            end_time: undefined,
            interval_second: '60',
            start_time: undefined,
          }
        },
        '10'
      );
    });

    it('builds trigger with a date if both start and end checkboxes are checked', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({ target: { type: 'checkbox', checked: true } });
      (tree.instance() as Trigger).handleChange('hasEndDate')({ target: { type: 'checkbox', checked: true } });
      expect(spy).toHaveBeenLastCalledWith(
        { periodic_schedule: { end_time: testDate, interval_second: '60', start_time: testDate } },
        '10'
      );
    });

    it('resets trigger to no start date if it is added then removed', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({ target: { type: 'checkbox', checked: true } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({ target: { type: 'checkbox', checked: false } });
      expect(spy).toHaveBeenLastCalledWith(
        { periodic_schedule: { end_time: undefined, interval_second: '60', start_time: undefined } },
        '10'
      );
    });

    it('builds trigger with a weekly interval', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('intervalCategory')({ target: { value: PeriodicInterval.WEEK } });
      expect(spy).toHaveBeenLastCalledWith(
        {
          periodic_schedule:
          {
            end_time: undefined,
            interval_second: (7 * 24 * 60 * 60).toString(),
            start_time: undefined
          }
        },
        '10'
      );
    });

    it('builds trigger with an every-three-months interval', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('intervalCategory')({ target: { value: PeriodicInterval.MONTH } });
      (tree.instance() as Trigger).handleChange('intervalValue')({ target: { value: 3 } });
      expect(spy).toHaveBeenLastCalledWith(
        {
          periodic_schedule:
          {
            end_time: undefined,
            interval_second: (3 * 30 * 24 * 60 * 60).toString(),
            start_time: undefined
          }
        },
        '10'
      );
    });

    it('builds trigger with the specified max concurrency setting', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.INTERVALED } });
      (tree.instance() as Trigger).handleChange('maxConcurrentRuns')({ target: { value: '3' } });
      expect(spy).toHaveBeenLastCalledWith(
        {
          periodic_schedule:
          {
            end_time: undefined,
            interval_second: '60',
            start_time: undefined
          }
        },
        '3'
      );
    });
  });

  describe('cron', () => {
    it('builds a 1-minute cron trigger by default', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      expect(spy).toHaveBeenLastCalledWith(
        { cron_schedule: { cron: '0 * * * * ?', end_time: undefined, start_time: undefined } },
        '10'
      );
    });

    it('builds a 1-minute cron trigger with specified start date', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({ target: { type: 'checkbox', checked: true } });
      (tree.instance() as Trigger).handleChange('startDate')({ target: { value: '2018-03-23' } });
      expect(spy).toHaveBeenLastCalledWith(
        { cron_schedule: { cron: '0 * * * * ?', end_time: undefined, start_time: testDate } },
        '10'
      );
    });

    it('builds a daily cron trigger with specified end date/time', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      (tree.instance() as Trigger).handleChange('hasEndDate')({ target: { type: 'checkbox', checked: true } });
      (tree.instance() as Trigger).handleChange('intervalCategory')({ target: { value: PeriodicInterval.DAY } });
      expect(spy).toHaveBeenLastCalledWith(
        { cron_schedule: { cron: '0 0 0 * * ?', end_time: testDate, start_time: undefined } },
        '10'
      );
    });

    it('builds a weekly cron trigger that runs every Monday, Friday, and Saturday', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      (tree.instance() as Trigger).handleChange('intervalCategory')({ target: { value: PeriodicInterval.WEEK } });
      (tree.instance() as any)._toggleCheckAllDays();
      (tree.instance() as any)._toggleDay(1);
      (tree.instance() as any)._toggleDay(5);
      (tree.instance() as any)._toggleDay(6);
      expect(spy).toHaveBeenLastCalledWith(
        { cron_schedule: { cron: '0 0 0 ? * 1,5,6', end_time: undefined, start_time: undefined } },
        '10'
      );
    });

    it('builds a cron with the manually specified cron string, even if days are toggled', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      (tree.instance() as Trigger).handleChange('intervalCategory')({ target: { value: PeriodicInterval.WEEK } });
      (tree.instance() as any)._toggleCheckAllDays();
      (tree.instance() as any)._toggleDay(1);
      (tree.instance() as any)._toggleDay(5);
      (tree.instance() as any)._toggleDay(6);
      (tree.instance() as Trigger).handleChange('editCron')({ target: { type: 'checkbox', checked: true } });
      (tree.instance() as Trigger).handleChange('cron')({ target: { value: 'oops this will break!' } });
      expect(spy).toHaveBeenLastCalledWith(
        { cron_schedule: { cron: 'oops this will break!', end_time: undefined, start_time: undefined } },
        '10'
      );
    });
  });
});
