// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import cronstrue from 'cronstrue';
import { TriggerSchedule } from './TriggerUtils';

/** Match robfig/cron v1: seconds first, optional trailing day-of-week. */
export function describeCron(expression: string): string {
  const cron = expression.trim();
  if (
    /^@every [+-]?(?:0|(?:([0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:ns|us|µs|μs|ms|s|m|h))+)$/.test(cron)
  ) {
    // The backend rounds these durations to whole seconds (minimum one second).
    return `Cron: ${cron}`;
  }
  const descriptors = [
    '@yearly',
    '@annually',
    '@monthly',
    '@weekly',
    '@daily',
    '@midnight',
    '@hourly',
  ];
  let normalized = cron;
  if (!descriptors.includes(cron)) {
    const fields = cron.split(/\s+/);
    if (fields.length < 5 || fields.length > 6 || /[^0-9a-zA-Z*?,/ -]/.test(cron))
      throw new Error('Use five or six cron fields, starting with seconds.');
    // Standard Unix five-field parsing would silently describe a different schedule.
    if (fields.length === 5) fields.push('*');
    const bounds = [
      [0, 59],
      [0, 59],
      [0, 23],
      [1, 31],
      [1, 12],
      [0, 6],
    ];
    const names: Record<string, number>[] = [
      {},
      {},
      {},
      {},
      Object.fromEntries(
        ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'].map(
          (name, i) => [name, i + 1],
        ),
      ),
      Object.fromEntries(
        ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT'].map((name, i) => [name, i]),
      ),
    ];
    fields.forEach((field, index) => {
      for (const part of field.split(',')) {
        const pieces = part.split('/');
        const [range, step] = pieces;
        if (pieces.length > 2 || (step !== undefined && (!/^\d+$/.test(step) || +step < 1)))
          throw new Error('Cron steps must be positive whole numbers.');
        if (range === '*' || range === '?') continue;
        const values = range.split('-').map((value) => {
          const number = /^\d+$/.test(value) ? +value : names[index][value.toUpperCase()];
          if (number === undefined || number < bounds[index][0] || number > bounds[index][1])
            throw new Error('A cron field is outside its allowed range.');
          return number;
        });
        if (values.length > 2 || (values.length === 2 && values[0] > values[1]))
          throw new Error('Cron ranges must run from lower to higher values.');
      }
    });
    // robfig combines restricted day-of-month and weekday fields with OR.
    if (!/[*?]/.test(fields[3]) && !/[*?]/.test(fields[5])) {
      const byDate = [...fields];
      byDate[5] = '*';
      const byWeekday = [...fields];
      byWeekday[3] = '*';
      return `${cronstrue.toString(byDate.join(' '), { use24HourTimeFormat: true })} or ${cronstrue.toString(byWeekday.join(' '), { use24HourTimeFormat: true })}`;
    }
    normalized = fields.join(' ');
  }
  try {
    return cronstrue.toString(normalized, { use24HourTimeFormat: true });
  } catch {
    throw new Error('Enter a valid cron expression.');
  }
}

/** Describe scheduler wall-clock times without converting to browser time. */
export function scheduleDescription(trigger?: TriggerSchedule): string {
  const cron = trigger?.cron_schedule?.cron;
  if (cron) {
    try {
      return `${describeCron(cron)} (scheduler time)`;
    } catch {
      return `Cron: ${cron}`;
    }
  }
  const seconds = Number(trigger?.periodic_schedule?.interval_second);
  if (!Number.isSafeInteger(seconds) || seconds <= 0) return '-';
  for (const [size, unit] of [
    [86400, 'day'],
    [3600, 'hour'],
    [60, 'minute'],
    [1, 'second'],
  ] as const) {
    if (seconds % size === 0) {
      const count = seconds / size;
      return count === 1 ? `Every ${unit}` : `Every ${count} ${unit}s`;
    }
  }
  return '-';
}
