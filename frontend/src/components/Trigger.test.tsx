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

const PARAMS_DEFAULT = {
  catchup: true,
  maxConcurrentRuns: '10',
};
const PERIODIC_DEFAULT = {
  end_time: undefined,
  interval_second: '60',
  start_time: undefined,
};
const CRON_DEFAULT = { cron: '0 * * * * ?', end_time: undefined, start_time: undefined };

beforeAll(() => {
  process.env.TZ = 'UTC';
});

describe('Trigger', () => {
  // tslint:disable-next-line:variable-name
  const RealDate = Date;

  function mockDate(isoDate: any): void {
    (global as any).Date = class extends RealDate {
      constructor(...args: any[]) {
        super();
        if (args.length === 0) {
          // Use mocked date when calling new Date()
          return new RealDate(isoDate);
        } else {
          // Otherwise, use real Date constructor
          return new (RealDate as any)(...args);
        }
      }
    };
  }
  const now = new Date(2018, 11, 21, 7, 53);
  mockDate(now);
  const oneWeekLater = new Date(2018, 11, 28, 7, 53);

  it('renders periodic schedule controls for initial render', () => {
    const tree = shallow(<Trigger t={(key: any) => key} />);
    expect(tree).toMatchSnapshot();
  });

  it('renders periodic schedule controls if the trigger type is CRON', () => {
    const tree = shallow(<Trigger t={(key: any) => key} />);
    (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
    expect(tree).toMatchSnapshot();
  });

  it('renders week days if the trigger type is CRON and interval is weekly', () => {
    const tree = shallow(<Trigger t={(key: any) => key} />);
    (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
    (tree.instance() as Trigger).handleChange('intervalCategory')({
      target: { value: PeriodicInterval.WEEK },
    });
    expect(tree).toMatchSnapshot();
  });

  it('renders all week days enabled', () => {
    const tree = shallow(<Trigger t={(key: any) => key} />);
    (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
    (tree.instance() as Trigger).handleChange('intervalCategory')({
      target: { value: PeriodicInterval.WEEK },
    });
    (tree.instance() as any)._toggleCheckAllDays();
    expect(tree).toMatchSnapshot();
  });

  it('enables a single day on click', () => {
    const tree = shallow(<Trigger t={(key: any) => key} />);
    (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
    (tree.instance() as Trigger).handleChange('intervalCategory')({
      target: { value: PeriodicInterval.WEEK },
    });
    (tree.instance() as any)._toggleDay(1);
    (tree.instance() as any)._toggleDay(3);
    expect(tree).toMatchSnapshot();
  });

  describe('interval trigger', () => {
    it('builds an every-minute trigger by default', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger with a start time if the checkbox is checked', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('hasStartDate')({
        target: { type: 'checkbox', checked: true },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: { ...PERIODIC_DEFAULT, start_time: now },
        },
      });
    });

    it('builds trigger with the entered start date/time', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('hasStartDate')({
        target: { type: 'checkbox', checked: true },
      });
      (tree.instance() as Trigger).handleChange('startDate')({ target: { value: '2018-11-23' } });
      (tree.instance() as Trigger).handleChange('startTime')({ target: { value: '08:35' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: {
            ...PERIODIC_DEFAULT,
            start_time: new Date(2018, 10, 23, 8, 35),
          },
        },
      });
    });

    it('builds trigger without the entered start date if no time is entered', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('hasStartDate')({
        target: { type: 'checkbox', checked: true },
      });
      (tree.instance() as Trigger).handleChange('startDate')({ target: { value: '2018-11-23' } });
      (tree.instance() as Trigger).handleChange('startTime')({ target: { value: '' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger without the entered start time if no date is entered', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('hasStartDate')({
        target: { type: 'checkbox', checked: true },
      });
      (tree.instance() as Trigger).handleChange('startDate')({ target: { value: '' } });
      (tree.instance() as Trigger).handleChange('startTime')({ target: { value: '11:33' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger with a date if both start and end checkboxes are checked', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('hasStartDate')({
        target: { type: 'checkbox', checked: true },
      });
      (tree.instance() as Trigger).handleChange('hasEndDate')({
        target: { type: 'checkbox', checked: true },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: { ...PERIODIC_DEFAULT, end_time: oneWeekLater, start_time: now },
        },
      });
    });

    it('resets trigger to no start date if it is added then removed', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('hasStartDate')({
        target: { type: 'checkbox', checked: true },
      });
      (tree.instance() as Trigger).handleChange('hasStartDate')({
        target: { type: 'checkbox', checked: false },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger with a weekly interval', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('intervalCategory')({
        target: { value: PeriodicInterval.WEEK },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: {
            ...PERIODIC_DEFAULT,
            interval_second: (7 * 24 * 60 * 60).toString(),
          },
        },
      });
    });

    it('builds trigger with an every-three-months interval', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('intervalCategory')({
        target: { value: PeriodicInterval.MONTH },
      });
      (tree.instance() as Trigger).handleChange('intervalValue')({ target: { value: 3 } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: {
            ...PERIODIC_DEFAULT,
            interval_second: (3 * 30 * 24 * 60 * 60).toString(),
          },
        },
      });
    });

    it('builds trigger with the specified max concurrency setting', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('maxConcurrentRuns')({ target: { value: '3' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        maxConcurrentRuns: '3',
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger with the specified catchup setting', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({
        target: { value: TriggerType.INTERVALED },
      });
      (tree.instance() as Trigger).handleChange('catchup')({
        target: { type: 'checkbox', checked: false },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        catchup: false,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('inits with cloned initial props', () => {
      const spy = jest.fn();
      const startTime = new Date('2020-01-01T23:53:00.000Z');
      shallow(
        <Trigger
          t={(key: any) => key}
          onChange={spy}
          initialProps={{
            maxConcurrentRuns: '3',
            catchup: false,
            trigger: {
              periodic_schedule: {
                interval_second: '' + 60 * 60 * 3, // 3 hours
                start_time: startTime.toISOString() as any,
              },
            },
          }}
        />,
      );
      expect(spy).toHaveBeenCalledTimes(1);
      expect(spy).toHaveBeenLastCalledWith({
        catchup: false,
        maxConcurrentRuns: '3',
        trigger: {
          periodic_schedule: {
            end_time: undefined,
            interval_second: '10800',
            start_time: startTime,
          },
        },
      });
    });
  });

  describe('cron', () => {
    it('builds a 1-minute cron trigger by default', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: CRON_DEFAULT,
        },
      });
    });

    it('builds a 1-minute cron trigger with specified start date', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      (tree.instance() as Trigger).handleChange('hasStartDate')({
        target: { type: 'checkbox', checked: true },
      });
      (tree.instance() as Trigger).handleChange('startDate')({ target: { value: '2018-03-23' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, start_time: new Date('2018-03-23T07:53:00.000Z') },
        },
      });
    });

    it('builds a daily cron trigger with specified end date/time', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      (tree.instance() as Trigger).handleChange('hasEndDate')({
        target: { type: 'checkbox', checked: true },
      });
      (tree.instance() as Trigger).handleChange('intervalCategory')({
        target: { value: PeriodicInterval.DAY },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, end_time: oneWeekLater, cron: '0 0 0 * * ?' },
        },
      });
    });

    it('builds a weekly cron trigger that runs every Monday, Friday, and Saturday', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      (tree.instance() as Trigger).handleChange('intervalCategory')({
        target: { value: PeriodicInterval.WEEK },
      });
      (tree.instance() as any)._toggleCheckAllDays();
      (tree.instance() as any)._toggleDay(1);
      (tree.instance() as any)._toggleDay(5);
      (tree.instance() as any)._toggleDay(6);
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, cron: '0 0 0 ? * 1,5,6' },
        },
      });
    });

    it('builds a cron with the manually specified cron string, even if days are toggled', () => {
      const spy = jest.fn();
      const tree = shallow(<Trigger t={(key: any) => key} onChange={spy} />);
      (tree.instance() as Trigger).handleChange('type')({ target: { value: TriggerType.CRON } });
      (tree.instance() as Trigger).handleChange('intervalCategory')({
        target: { value: PeriodicInterval.WEEK },
      });
      (tree.instance() as any)._toggleCheckAllDays();
      (tree.instance() as any)._toggleDay(1);
      (tree.instance() as any)._toggleDay(5);
      (tree.instance() as any)._toggleDay(6);
      (tree.instance() as Trigger).handleChange('editCron')({
        target: { type: 'checkbox', checked: true },
      });
      (tree.instance() as Trigger).handleChange('cron')({
        target: { value: 'oops this will break!' },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, cron: 'oops this will break!' },
        },
      });
    });

    it('inits with cloned initial props', () => {
      const spy = jest.fn();
      const startTime = new Date('2020-01-01T00:00:00.000Z');
      const endTime = new Date('2020-01-02T01:02:00.000Z');
      shallow(
        <Trigger
          t={(key: any) => key}
          onChange={spy}
          initialProps={{
            maxConcurrentRuns: '4',
            catchup: true,
            trigger: {
              cron_schedule: {
                cron: '0 0 0 ? * 1,5,6',
                start_time: startTime.toISOString() as any,
                end_time: endTime.toISOString() as any,
              },
            },
          }}
        />,
      );
      expect(spy).toHaveBeenCalledTimes(1);
      expect(spy).toHaveBeenLastCalledWith({
        catchup: true,
        maxConcurrentRuns: '4',
        trigger: {
          cron_schedule: {
            cron: '0 0 0 ? * 1,5,6',
            start_time: startTime,
            end_time: endTime,
          },
        },
      });
    });
  });
});
