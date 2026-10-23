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

import { describeCron, scheduleDescription } from './ScheduleDisplay';

describe('scheduleDescription', () => {
  it.each([
    ['1', 'Every second'],
    ['60', 'Every minute'],
    ['3600', 'Every hour'],
    ['86400', 'Every day'],
    ['900', 'Every 15 minutes'],
    ['7200', 'Every 2 hours'],
    ['0', '-'],
    ['invalid', '-'],
    ['-1', '-'],
  ])('formats interval %s', (interval_second, expected) => {
    expect(scheduleDescription({ periodic_schedule: { interval_second } })).toBe(expected);
  });
  it.each([
    ['0 0 9 * * *', 'At 09:00 (scheduler time)'],
    ['0 30 8 * * MON-FRI', 'At 08:30, Monday through Friday (scheduler time)'],
    ['*/15 * * * *', 'Every 15 seconds (scheduler time)'],
    ['CRON_TZ=America/New_York 0 0 9 * * *', 'Cron: CRON_TZ=America/New_York 0 0 9 * * *'],
    ['bad cron', 'Cron: bad cron'],
  ])('describes cron %s', (cron, expected) => {
    expect(scheduleDescription({ cron_schedule: { cron } })).toBe(expected);
  });
  it('handles missing schedules', () => expect(scheduleDescription()).toBe('-'));
});

describe('backend cron semantics', () => {
  it.each([
    '*/0 * * * * *',
    '*/1.5 * * * * *',
    '*/2/3 * * * * *',
    '30-10 * * * * *',
    '0 0 9 * * FRI-MON',
    '0 0 9 * DEC-JAN *',
  ])('rejects invalid steps or descending ranges: %s', (cron) => {
    expect(() => describeCron(cron)).toThrow();
    expect(scheduleDescription({ cron_schedule: { cron } })).toBe(`Cron: ${cron}`);
  });

  it('treats five fields as seconds first with the trailing weekday omitted', () => {
    expect(describeCron('*/15 * * * *')).toBe('Every 15 seconds');
    expect(describeCron('0 30 8 * *')).toBe('At 08:30');
  });

  it('describes restricted day-of-month and weekday as alternatives', () => {
    const description = describeCron('0 0 9 1 * MON');
    expect(description).toBe(`${describeCron('0 0 9 1 * *')} or ${describeCron('0 0 9 * * MON')}`);
    expect(description).toContain('day 1');
    expect(description).toContain('Monday');
  });

  it.each(['@every .5h', '@every +1h', '@every 0', '@every 0.5s'])(
    'preserves Go duration expression %s without claiming a subsecond cadence',
    (cron) => {
      expect(describeCron(cron)).toBe(`Cron: ${cron}`);
    },
  );
});
