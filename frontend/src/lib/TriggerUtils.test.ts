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

import {
  getPeriodInSeconds,
  PeriodicInterval,
  buildCron,
  pickersToDate,
  buildTrigger,
  TriggerType,
  dateToPickerFormat,
  triggerDisplayString,
  parseTrigger,
} from './TriggerUtils';
import { ApiTrigger } from '../apis/job';

describe('TriggerUtils', () => {
  describe('getPeriodInSeconds', () => {
    it('returns minutes to seconds', () => {
      expect(getPeriodInSeconds(PeriodicInterval.MINUTE, 0)).toBe(0);
      expect(getPeriodInSeconds(PeriodicInterval.MINUTE, 1)).toBe(60);
      expect(getPeriodInSeconds(PeriodicInterval.MINUTE, 17)).toBe(17 * 60);
    });

    it('returns hours to seconds', () => {
      expect(getPeriodInSeconds(PeriodicInterval.HOUR, 0)).toBe(0);
      expect(getPeriodInSeconds(PeriodicInterval.HOUR, 1)).toBe(60 * 60);
      expect(getPeriodInSeconds(PeriodicInterval.HOUR, 17)).toBe(17 * 60 * 60);
    });

    it('returns days to seconds', () => {
      expect(getPeriodInSeconds(PeriodicInterval.DAY, 0)).toBe(0);
      expect(getPeriodInSeconds(PeriodicInterval.DAY, 1)).toBe(24 * 60 * 60);
      expect(getPeriodInSeconds(PeriodicInterval.DAY, 17)).toBe(17 * 24 * 60 * 60);
    });

    it('returns weeks to seconds', () => {
      expect(getPeriodInSeconds(PeriodicInterval.WEEK, 0)).toBe(0);
      expect(getPeriodInSeconds(PeriodicInterval.WEEK, 1)).toBe(7 * 24 * 60 * 60);
      expect(getPeriodInSeconds(PeriodicInterval.WEEK, 17)).toBe(17 * 7 * 24 * 60 * 60);
    });

    it('returns months to seconds', () => {
      expect(getPeriodInSeconds(PeriodicInterval.MONTH, 0)).toBe(0);
      expect(getPeriodInSeconds(PeriodicInterval.MONTH, 1)).toBe(30 * 24 * 60 * 60);
      expect(getPeriodInSeconds(PeriodicInterval.MONTH, 17)).toBe(17 * 30 * 24 * 60 * 60);
    });

    it('works with negative numbers', () => {
      expect(getPeriodInSeconds(PeriodicInterval.HOUR, -17)).toBe(-17 * 60 * 60);
    });

    it('throws on bad interval category', () => {
      expect(() => getPeriodInSeconds(88 as any, 1)).toThrowError('Invalid interval category: 88');
    });
  });

  describe('buildCron', () => {
    it('builds (every minute) cron without start date, no selected days', () => {
      expect(buildCron(undefined, PeriodicInterval.MINUTE, [])).toBe('0 * * * * ?');
    });

    it('builds (every hour) cron without start date, no selected days', () => {
      expect(buildCron(undefined, PeriodicInterval.HOUR, [])).toBe('0 0 * * * ?');
    });

    it('builds (every day) cron without start date, no selected days', () => {
      expect(buildCron(undefined, PeriodicInterval.DAY, [])).toBe('0 0 0 * * ?');
    });

    it('builds (every week) cron without start date, no selected days', () => {
      expect(buildCron(undefined, PeriodicInterval.WEEK, [])).toBe('0 0 0 ? * *');
    });

    it('builds (every month) cron without start date, no selected days', () => {
      expect(buildCron(undefined, PeriodicInterval.MONTH, [])).toBe('0 0 0 0 * ?');
    });

    const date = new Date(2018, 1, 13, 8, 53);
    it('adds the given minutes to (every hour), no selected days', () => {
      expect(buildCron(date, PeriodicInterval.HOUR, [])).toBe('0 53 * * * ?');
    });

    it('adds the given hour+minute (every day), no selected days', () => {
      expect(buildCron(date, PeriodicInterval.DAY, [])).toBe('0 53 8 * * ?');
    });

    it('adds the given day+hour+minute (every week), no selected days', () => {
      expect(buildCron(date, PeriodicInterval.WEEK, [])).toBe('0 53 8 ? * *');
    });

    it('adds the given hour+minute (every month), no selected days', () => {
      expect(buildCron(date, PeriodicInterval.MONTH, [])).toBe('0 53 8 13 * ?');
    });

    function setToSelectedDays(selected: Set<number>): boolean[] {
      return new Array(7).fill(false).map((_, i) => (selected.has(i) ? true : false));
    }

    it('adds no days when building weekly cron with no selected days, no star date', () => {
      expect(buildCron(undefined, PeriodicInterval.WEEK, setToSelectedDays(new Set()))).toBe(
        '0 0 0 ? *',
      );
    });

    it('adds all days when building weekly cron, all days selected, no star date', () => {
      expect(
        buildCron(
          undefined,
          PeriodicInterval.WEEK,
          setToSelectedDays(new Set([0, 1, 2, 3, 4, 5, 6])),
        ),
      ).toBe('0 0 0 ? * *');
    });

    it('adds selected days when building weekly cron, some days selected, no start date', () => {
      expect(
        buildCron(undefined, PeriodicInterval.WEEK, setToSelectedDays(new Set([0, 3, 4, 5]))),
      ).toBe('0 0 0 ? * 0,3,4,5');
    });

    it('adds selected days when building weekly cron, some days selected, no start date', () => {
      expect(buildCron(undefined, PeriodicInterval.WEEK, setToSelectedDays(new Set([0, 1])))).toBe(
        '0 0 0 ? * 0,1',
      );
    });

    it('adds selected days when building weekly cron, some days selected, with start date', () => {
      expect(buildCron(date, PeriodicInterval.WEEK, setToSelectedDays(new Set([0, 1])))).toBe(
        '0 53 8 ? * 0,1',
      );
    });

    it('throws on bad interval category value', () => {
      expect(() => buildCron(undefined, 88 as any, [])).toThrow();
    });
  });

  describe('pickersToDate', () => {
    it('returns undefined if hasDate is false', () => {
      expect(pickersToDate(false, 'some string', 'some string')).toBeUndefined();
    });

    // JS dates are 0-indexed, so 0 here is actually January.
    const date = new Date(2018, 0, 13, 17, 33);
    it('converts picker format to date if hasDate is true', () => {
      expect(pickersToDate(true, '2018-1-13', '17:33')!.toISOString()).toBe(date.toISOString());
    });

    it('throws on bad format', () => {
      expect(() => pickersToDate(true, '2018/8/13', '17:33')).toThrow();
    });
  });

  describe('buildTrigger', () => {
    it('returns a cron trigger object if type is so set', () => {
      expect(
        buildTrigger(PeriodicInterval.DAY, 1, undefined, undefined, TriggerType.CRON, 'some cron'),
      ).toEqual({
        cron_schedule: { cron: 'some cron' },
      });
    });

    it('returns an interval trigger object if type is so set', () => {
      expect(
        buildTrigger(PeriodicInterval.MINUTE, 1, undefined, undefined, TriggerType.INTERVALED, ''),
      ).toEqual({
        periodic_schedule: { interval_second: '60' },
      });
    });

    it('returns an interval trigger object if type is so set, and a cron string is passed', () => {
      expect(
        buildTrigger(
          PeriodicInterval.MINUTE,
          1,
          undefined,
          undefined,
          TriggerType.INTERVALED,
          'some cron',
        ),
      ).toEqual({
        periodic_schedule: { interval_second: '60' },
      });
    });

    const startDate = new Date(2018, 8, 13, 17, 33);
    const endDate = new Date(2018, 8, 17, 17, 33);
    it('adds start date/time to returned cron trigger', () => {
      expect(
        buildTrigger(PeriodicInterval.MINUTE, 1, startDate, endDate, TriggerType.CRON, 'some cron'),
      ).toEqual({
        cron_schedule: {
          cron: 'some cron',
          end_time: endDate,
          start_time: startDate,
        },
      });
    });

    it('adds start date/time to returned interval trigger', () => {
      expect(
        buildTrigger(PeriodicInterval.MINUTE, 1, startDate, endDate, TriggerType.INTERVALED, ''),
      ).toEqual({
        periodic_schedule: {
          end_time: endDate,
          interval_second: '60',
          start_time: startDate,
        },
      });
    });

    it('throws if passed an unrecognized trigger type', () => {
      expect(() =>
        buildTrigger(
          PeriodicInterval.DAY,
          1,
          undefined,
          undefined,
          'not a trigger type' as any,
          '',
        ),
      ).toThrowError('Invalid TriggerType: ' + 'not a trigger type');
    });
  });

  describe('parseTrigger', () => {
    it('throws on invalid trigger', () => {
      expect(() => parseTrigger({})).toThrow('Invalid trigger: {}');
    });

    it('parses periodic schedule', () => {
      const startTime = new Date(1234);
      const parsedTrigger = parseTrigger({
        periodic_schedule: {
          start_time: startTime,
          interval_second: '120',
        },
      });
      expect(parsedTrigger).toEqual({
        type: TriggerType.INTERVALED,
        startDateTime: startTime,
        endDateTime: undefined,
        intervalCategory: PeriodicInterval.MINUTE,
        intervalValue: 2,
      });
    });

    it('parses cron schedule', () => {
      const endTime = new Date(12345);
      const parsedTrigger = parseTrigger({
        cron_schedule: {
          end_time: endTime,
          cron: '0 0 0 ? * 0,6',
        },
      });
      expect(parsedTrigger).toEqual({
        type: TriggerType.CRON,
        cron: '0 0 0 ? * 0,6',
        startDateTime: undefined,
        endDateTime: endTime,
      });
    });
  });

  describe('dateToPickerFormat', () => {
    it('converts date to picker format date and time', () => {
      const testDate = new Date(2018, 11, 13, 11, 33);
      const [date, time] = dateToPickerFormat(testDate);
      expect(date).toBe('2018-12-13');
      expect(time).toBe('11:33');
    });

    it('pads date and time if they are single digit', () => {
      const testDate = new Date(2018, 0, 3, 2, 5);
      const [date, time] = dateToPickerFormat(testDate);
      expect(date).toBe('2018-01-03');
      expect(time).toBe('02:05');
    });
  });

  describe('triggerDisplayString', () => {
    it('handles no trigger', () => {
      expect(triggerDisplayString()).toBe('-');
    });

    it('returns the cron string for a cron trigger', () => {
      expect(triggerDisplayString({ cron_schedule: { cron: 'some cron' } })).toBe('some cron');
    });

    const testTrigger: ApiTrigger = { periodic_schedule: {} };
    it('handles a periodic trigger with no body', () => {
      expect(triggerDisplayString(testTrigger)).toBe('-');
    });

    it('handles a periodic trigger with dates but no interval', () => {
      const trigger = { ...testTrigger } as any;
      trigger.periodic_schedule.start_time = new Date();
      expect(triggerDisplayString()).toBe('-');
    });

    it('says never if the interval is zero', () => {
      const trigger: ApiTrigger = { periodic_schedule: { interval_second: '0' } };
      expect(triggerDisplayString(trigger)).toBe('Never');
    });

    it('uses seconds for less than a minute', () => {
      const trigger: ApiTrigger = { periodic_schedule: { interval_second: '1' } };
      expect(triggerDisplayString(trigger)).toBe('Every 1 seconds');
    });

    it('uses seconds for less than a minute', () => {
      const trigger: ApiTrigger = { periodic_schedule: { interval_second: '59' } };
      expect(triggerDisplayString(trigger)).toBe('Every 59 seconds');
    });

    it('uses minutes for less than an hour', () => {
      const trigger: ApiTrigger = { periodic_schedule: { interval_second: '60' } };
      expect(triggerDisplayString(trigger)).toBe('Every 1 minutes');
    });

    it('uses minutes for less than an hour', () => {
      const trigger: ApiTrigger = { periodic_schedule: { interval_second: '61' } };
      expect(triggerDisplayString(trigger)).toBe('Every 1 minutes, and 1 seconds');
    });

    it('uses minutes for less than an hour', () => {
      const trigger: ApiTrigger = {
        periodic_schedule: { interval_second: (60 * 60 - 1).toString() },
      };
      expect(triggerDisplayString(trigger)).toBe('Every 59 minutes, and 59 seconds');
    });

    it('uses hours for less than a day', () => {
      const trigger: ApiTrigger = { periodic_schedule: { interval_second: (60 * 60).toString() } };
      expect(triggerDisplayString(trigger)).toBe('Every 1 hours');
    });

    it('uses hours for less than a day', () => {
      const trigger: ApiTrigger = {
        periodic_schedule: { interval_second: (60 * 60 + 1).toString() },
      };
      expect(triggerDisplayString(trigger)).toBe('Every 1 hours, and 1 seconds');
    });

    it('uses hours for less than a day', () => {
      const trigger: ApiTrigger = {
        periodic_schedule: { interval_second: (2 * 60 * 60 + 63).toString() },
      };
      expect(triggerDisplayString(trigger)).toBe('Every 2 hours, 1 minutes, and 3 seconds');
    });

    it('uses days for less than a week', () => {
      const trigger: ApiTrigger = {
        periodic_schedule: { interval_second: (24 * 60 * 60).toString() },
      };
      expect(triggerDisplayString(trigger)).toBe('Every 1 days');
    });

    it('uses days for less than a week', () => {
      const trigger: ApiTrigger = {
        periodic_schedule: { interval_second: (24 * 60 * 60 + 1).toString() },
      };
      expect(triggerDisplayString(trigger)).toBe('Every 1 days, and 1 seconds');
    });

    it('uses weeks for less than a month', () => {
      const trigger: ApiTrigger = {
        periodic_schedule: { interval_second: (7 * 24 * 60 * 60).toString() },
      };
      expect(triggerDisplayString(trigger)).toBe('Every 1 weeks');
    });

    it('uses weeks for less than a month', () => {
      const trigger: ApiTrigger = {
        periodic_schedule: { interval_second: (8 * 24 * 60 * 60 + 63).toString() },
      };
      expect(triggerDisplayString(trigger)).toBe('Every 1 weeks, 1 days, 1 minutes, and 3 seconds');
    });

    it('uses months for anything large', () => {
      const trigger: ApiTrigger = {
        periodic_schedule: { interval_second: (80 * 24 * 60 * 60).toString() },
      };
      expect(triggerDisplayString(trigger)).toBe('Every 2 months, 2 weeks, and 3 days');
    });
  });
});
