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

import { ApiTrigger, ApiPeriodicSchedule, ApiCronSchedule } from '../../src/apis/job';

export enum TriggerType {
  INTERVALED,
  CRON,
}

export enum PeriodicInterval {
  MINUTE = 'experiments:periodicIntervalEnum.minute',
  HOUR = 'experiments:periodicIntervalEnum.hour',
  DAY = 'experiments:periodicIntervalEnum.day',
  WEEK = 'experiments:periodicIntervalEnum.week',
  MONTH = 'experiments:periodicIntervalEnum.month',
}

const INTERVAL_SECONDS = {
  [PeriodicInterval.MINUTE]: 60,
  [PeriodicInterval.HOUR]: 60 * 60,
  [PeriodicInterval.DAY]: 60 * 60 * 24,
  [PeriodicInterval.WEEK]: 60 * 60 * 24 * 7,
  [PeriodicInterval.MONTH]: 60 * 60 * 24 * 30,
};
const PERIODIC_INTERVAL_DESCENDING = [
  PeriodicInterval.MONTH,
  PeriodicInterval.WEEK,
  PeriodicInterval.DAY,
  PeriodicInterval.HOUR,
  PeriodicInterval.MINUTE,
];

export const triggers = new Map<TriggerType, { displayName: string }>([
  [TriggerType.INTERVALED, { displayName: 'experiments:triggerTypeEnum.periodic' }],
  [TriggerType.CRON, { displayName: 'experiments:triggerTypeEnum.cron' }],
]);

export function getPeriodInSeconds(interval: PeriodicInterval, count: number): number {
  const intervalSeconds = INTERVAL_SECONDS[interval];
  if (!intervalSeconds) {
    throw new Error('Invalid interval category: ' + interval);
  }
  return intervalSeconds * count;
}
export function parsePeriodFromSeconds(
  seconds: number,
): { interval: PeriodicInterval; count: number } {
  for (const interval of PERIODIC_INTERVAL_DESCENDING) {
    const intervalSeconds = INTERVAL_SECONDS[interval];
    if (seconds % intervalSeconds === 0) {
      return {
        interval,
        count: seconds / intervalSeconds,
      };
    }
  }
  throw new Error('Invalid seconds: ' + seconds);
}

export function buildCron(
  startDateTime: Date | undefined,
  intervalCategory: PeriodicInterval,
  selectedDays: boolean[],
): string {
  const isAllDaysChecked = selectedDays.every(d => !!d);
  let targetDayOfMonth = '0';
  let targetHours = '0';
  let targetMinutes = '0';
  if (startDateTime) {
    targetDayOfMonth = '' + startDateTime.getDate();
    targetHours = '' + startDateTime.getHours();
    targetMinutes = '' + startDateTime.getMinutes();
  }

  // The default values here correspond to 'run at second 0 of every minute'
  const second = '0';
  let minute = '*';
  let hour = '*';
  let dayOfMonth = '*';
  const month = '*';
  let dayOfWeek = '?';
  switch (intervalCategory) {
    case PeriodicInterval.MINUTE:
      break;
    case PeriodicInterval.HOUR:
      minute = targetMinutes;
      break;
    case PeriodicInterval.DAY:
      minute = targetMinutes;
      hour = targetHours;
      break;
    case PeriodicInterval.WEEK:
      minute = targetMinutes;
      hour = targetHours;
      dayOfMonth = '?';
      if (isAllDaysChecked) {
        dayOfWeek = '*';
      } else {
        // Convert weekdays to array of indices of active days and join them.
        dayOfWeek = selectedDays
          .reduce((result: number[], day, i) => {
            if (day) {
              result.push(i);
            }
            return result;
          }, [])
          .join(',');
      }
      break;
    case PeriodicInterval.MONTH:
      minute = targetMinutes;
      hour = targetHours;
      dayOfMonth = targetDayOfMonth;
      break;
    default:
      throw new Error('Invalid interval index: ' + intervalCategory);
  }

  return [second, minute, hour, dayOfMonth, month, dayOfWeek].join(' ').trim();
}

export function pickersToDate(
  hasDate: boolean,
  dateStr: string,
  timeStr: string,
): Date | undefined {
  if (hasDate && dateStr && timeStr) {
    const [year, month, date] = dateStr.split('-');
    const [hour, minute] = timeStr.split(':');

    const d = new Date(+year, +month - 1, +date, +hour, +minute);
    if (isNaN(d as any)) {
      throw new Error('Invalid picker format');
    }
    return d;
  } else {
    return undefined;
  }
}

export function buildTrigger(
  intervalCategory: PeriodicInterval,
  intervalValue: number,
  startDateTime: Date | undefined,
  endDateTime: Date | undefined,
  type: TriggerType,
  cron: string,
): ApiTrigger {
  let trigger: ApiTrigger;
  switch (type) {
    case TriggerType.INTERVALED:
      trigger = {
        periodic_schedule: {
          interval_second: getPeriodInSeconds(intervalCategory, intervalValue).toString(),
        } as ApiPeriodicSchedule,
      };
      trigger.periodic_schedule!.start_time = startDateTime;
      trigger.periodic_schedule!.end_time = endDateTime;
      break;
    case TriggerType.CRON:
      trigger = {
        cron_schedule: {
          cron,
        } as ApiCronSchedule,
      };
      trigger.cron_schedule!.start_time = startDateTime;
      trigger.cron_schedule!.end_time = endDateTime;
      break;
    default:
      throw new Error(`Invalid TriggerType: ${type}`);
  }

  return trigger;
}

export type ParsedTrigger =
  | {
      type: TriggerType.INTERVALED;
      intervalCategory: PeriodicInterval;
      intervalValue: number;
      startDateTime?: Date;
      endDateTime?: Date;
      cron?: undefined;
    }
  | {
      type: TriggerType.CRON;
      intervalCategory?: undefined;
      intervalValue?: undefined;
      startDateTime?: Date;
      endDateTime?: Date;
      cron: string;
    };

export function parseTrigger(trigger: ApiTrigger): ParsedTrigger {
  if (trigger.periodic_schedule) {
    const periodicSchedule = trigger.periodic_schedule;
    const intervalSeconds = parseInt(periodicSchedule.interval_second || '', 10);
    if (Number.isNaN(intervalSeconds)) {
      throw new Error(
        `Interval seconds is NaN: ${periodicSchedule.interval_second} for ${JSON.stringify(
          trigger,
        )}`,
      );
    }
    const { interval: intervalCategory, count: intervalValue } = parsePeriodFromSeconds(
      intervalSeconds,
    );
    return {
      type: TriggerType.INTERVALED,
      intervalCategory,
      intervalValue,
      // Generated client has a bug the fields will be string here instead, so
      // we use new Date() to convert them to Date.
      startDateTime: periodicSchedule.start_time
        ? new Date(periodicSchedule.start_time as any)
        : undefined,
      endDateTime: periodicSchedule.end_time
        ? new Date(periodicSchedule.end_time as any)
        : undefined,
    };
  }
  if (trigger.cron_schedule) {
    const { cron, start_time: startTime, end_time: endTime } = trigger.cron_schedule;
    return {
      type: TriggerType.CRON,
      cron: cron || '',
      // Generated client has a bug the fields will be string here instead, so
      // we use new Date() to convert them to Date.
      startDateTime: startTime ? new Date(startTime as any) : undefined,
      endDateTime: endTime ? new Date(endTime as any) : undefined,
    };
  }
  throw new Error(`Invalid trigger: ${JSON.stringify(trigger)}`);
}

export function dateToPickerFormat(d: Date): [string, string] {
  const year = d.getFullYear();
  const month = ('0' + (d.getMonth() + 1)).slice(-2);
  const date = ('0' + d.getDate()).slice(-2);
  const hour = ('0' + d.getHours()).slice(-2);
  const minute = ('0' + d.getMinutes()).slice(-2);

  const nowDate = year + '-' + month + '-' + date;
  const nowTime = hour + ':' + minute;

  return [nowDate, nowTime];
}

export function triggerDisplayString(trigger?: ApiTrigger): string {
  if (trigger) {
    if (trigger.cron_schedule && trigger.cron_schedule.cron) {
      return trigger.cron_schedule.cron;
    }
    if (trigger.periodic_schedule && trigger.periodic_schedule.interval_second) {
      const intervalSecond = trigger.periodic_schedule.interval_second;
      if (intervalSecond === '0') {
        return 'Never';
      }
      const secInMin = 60;
      const secInHour = secInMin * 60;
      const secInDay = secInHour * 24;
      const secInWeek = secInDay * 7;
      const secInMonth = secInDay * 30;
      const months = Math.floor(+intervalSecond / secInMonth);
      const weeks = Math.floor((+intervalSecond % secInMonth) / secInWeek);
      const days = Math.floor((+intervalSecond % secInWeek) / secInDay);
      const hours = Math.floor((+intervalSecond % secInDay) / secInHour);
      const minutes = Math.floor((+intervalSecond % secInHour) / secInMin);
      const seconds = Math.floor(+intervalSecond % secInMin);
      let interval = 'Every';

      if (months) {
        interval += ` ${months} months,`;
      }
      if (weeks) {
        interval += ` ${weeks} weeks,`;
      }
      if (days) {
        interval += ` ${days} days,`;
      }
      if (hours) {
        interval += ` ${hours} hours,`;
      }
      if (minutes) {
        interval += ` ${minutes} minutes,`;
      }
      if (seconds) {
        interval += ` ${seconds} seconds,`;
      }
      // Add 'and' if necessary
      const insertAndLocation = interval.lastIndexOf(', ') + 1;
      if (insertAndLocation > 0) {
        interval =
          interval.slice(0, insertAndLocation) + ' and' + interval.slice(insertAndLocation);
      }
      // Remove trailing comma
      return interval.slice(0, -1);
    }
  }
  return '-';
}
