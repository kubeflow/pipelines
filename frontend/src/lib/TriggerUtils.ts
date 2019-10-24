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
  MINUTE = 'Minute',
  HOUR = 'Hour',
  DAY = 'Day',
  WEEK = 'Week',
  MONTH = 'Month',
}

export const triggers = new Map<TriggerType, { displayName: string }>([
  [TriggerType.INTERVALED, { displayName: 'Periodic' }],
  [TriggerType.CRON, { displayName: 'Cron' }],
]);

export function getPeriodInSeconds(interval: PeriodicInterval, count: number): number {
  let intervalSeconds = 0;
  switch (interval) {
    case PeriodicInterval.MINUTE:
      intervalSeconds = 60;
      break;
    case PeriodicInterval.HOUR:
      intervalSeconds = 60 * 60;
      break;
    case PeriodicInterval.DAY:
      intervalSeconds = 60 * 60 * 24;
      break;
    case PeriodicInterval.WEEK:
      intervalSeconds = 60 * 60 * 24 * 7;
      break;
    case PeriodicInterval.MONTH:
      intervalSeconds = 60 * 60 * 24 * 30;
      break;
    default:
      throw new Error('Invalid interval category: ' + interval);
  }
  return intervalSeconds * count;
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
