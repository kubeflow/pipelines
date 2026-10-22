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

import * as React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import Trigger from './Trigger';

const defaults = { catchup: true, maxConcurrentRuns: '10' };
const periodic = { interval_second: '3600', start_time: undefined, end_time: undefined };

function change(label: string, value: string) {
  fireEvent.change(screen.getByLabelText(label, { exact: true }), { target: { value } });
}

function openAdvanced() {
  fireEvent.click(screen.getByText('Advanced execution settings', { exact: true }));
}

describe('Trigger schedule editor', () => {
  it('defaults to an hourly interval and hides optional boundaries', () => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    expect(screen.getByLabelText('Repeat')).toHaveValue('interval');
    expect(screen.queryByLabelText('Start date')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('End date')).not.toBeInTheDocument();
    expect(onChange).toHaveBeenLastCalledWith({
      ...defaults,
      trigger: { periodic_schedule: periodic },
    });
  });

  it('keeps fixed intervals separate from calendar schedules', () => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    change('Interval', '3');
    change('Interval unit', 'Hour');
    expect(onChange).toHaveBeenLastCalledWith({
      ...defaults,
      trigger: {
        periodic_schedule: { ...periodic, interval_second: '10800' },
      },
    });
    expect(screen.queryByRole('option', { name: 'Months' })).not.toBeInTheDocument();
  });

  it.each([
    ['daily', '0 0 9 * * *'],
    ['weekly', '0 0 9 * * 1,2,3,4,5'],
    ['monthly', '0 0 9 1 * *'],
  ])('builds the %s preset as a calendar schedule', (repeat, cron) => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    change('Repeat', repeat);
    expect(onChange).toHaveBeenLastCalledWith({
      ...defaults,
      trigger: {
        cron_schedule: { cron, start_time: undefined, end_time: undefined },
      },
    });
  });

  it('updates calendar time and day independently of boundaries', () => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    change('Repeat', 'monthly');
    change('Day of month', '15');
    change('Run at', '14:30');
    expect(onChange).toHaveBeenLastCalledWith({
      ...defaults,
      trigger: {
        cron_schedule: { cron: '0 30 14 15 * *', start_time: undefined, end_time: undefined },
      },
    });
  });

  it('supports weekday selection and rejects an empty selection', () => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    change('Repeat', 'weekly');
    for (const day of ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday']) {
      fireEvent.click(screen.getByRole('button', { name: day, exact: true }));
    }
    expect(onChange).toHaveBeenLastCalledWith({ ...defaults, trigger: undefined });
    fireEvent.click(screen.getByRole('button', { name: 'Saturday', exact: true }));
    expect(onChange).toHaveBeenLastCalledWith({
      ...defaults,
      trigger: {
        cron_schedule: { cron: '0 0 9 * * 6', start_time: undefined, end_time: undefined },
      },
    });
  });

  it('validates custom cron and recovers without silently changing it', () => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    change('Repeat', 'custom');
    change('Cron expression', 'invalid cron');
    expect(onChange).toHaveBeenLastCalledWith({ ...defaults, trigger: undefined });
    change('Cron expression', '0 15 8 ? * 1-5');
    expect(onChange).toHaveBeenLastCalledWith({
      ...defaults,
      trigger: {
        cron_schedule: { cron: '0 15 8 ? * 1-5', start_time: undefined, end_time: undefined },
      },
    });
  });

  it.each(['0', '-1', '1.5'])('rejects invalid interval %s', (value) => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    change('Interval unit', 'Second');
    change('Interval', value);
    expect(onChange).toHaveBeenLastCalledWith({ ...defaults, trigger: undefined });
  });

  it('accepts fractional hours that produce whole seconds', () => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    change('Interval', '1.5');
    expect(onChange).toHaveBeenLastCalledWith({
      ...defaults,
      trigger: { periodic_schedule: { ...periodic, interval_second: '5400' } },
    });
  });

  it('keeps concurrency and catchup settings in advanced options', () => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    openAdvanced();
    change('Maximum concurrent runs', '3');
    fireEvent.click(
      screen.getByRole('checkbox', { name: 'Run missed schedules when resumed', exact: true }),
    );
    expect(onChange).toHaveBeenLastCalledWith({
      catchup: false,
      maxConcurrentRuns: '3',
      trigger: { periodic_schedule: periodic },
    });
  });

  it.each(['0', '-1', '1.5', '11', 'abc'])('rejects invalid concurrency %s', (value) => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    openAdvanced();
    change('Maximum concurrent runs', value);
    expect(onChange.mock.lastCall?.[0].maxConcurrentRuns).toBeUndefined();
  });

  it.each(['Start', 'End'])(
    'validates %s boundaries and removes them when deselected',
    (boundary) => {
      const onChange = vi.fn();
      render(<Trigger onChange={onChange} />);
      change(boundary === 'Start' ? 'Starts' : 'Ends', 'date');
      change(`${boundary} date`, '2026-10-20');
      change(`${boundary} time`, '');
      expect(onChange).toHaveBeenLastCalledWith({ ...defaults, trigger: undefined });
      expect(screen.getByLabelText(`${boundary} time`)).toBeInvalid();
      change(`${boundary} time`, '08:35');
      expect(screen.getByLabelText(`${boundary} time`)).toBeValid();
      change(boundary === 'Start' ? 'Starts' : 'Ends', boundary === 'Start' ? 'now' : 'never');
      expect(onChange).toHaveBeenLastCalledWith({
        ...defaults,
        trigger: { periodic_schedule: periodic },
      });
    },
  );

  it('converts chosen local boundaries to instants and rejects reversed windows', () => {
    const onChange = vi.fn();
    render(<Trigger onChange={onChange} />);
    change('Starts', 'date');
    change('Start date', '2026-10-20');
    change('Start time', '08:35');
    change('Ends', 'date');
    change('End date', '2026-10-21');
    change('End time', '18:45');
    expect(onChange).toHaveBeenLastCalledWith({
      ...defaults,
      trigger: {
        periodic_schedule: {
          ...periodic,
          start_time: new Date(2026, 9, 20, 8, 35),
          end_time: new Date(2026, 9, 21, 18, 45),
        },
      },
    });
    change('End date', '2026-10-19');
    expect(onChange).toHaveBeenLastCalledWith({ ...defaults, trigger: undefined });
  });

  it.each(['17 15 8 ? * 1-5', '@every 90s', '@every .5h', '@every +1h', '@every 0'])(
    'preserves existing cron %s and timestamp precision on unrelated changes',
    (cron) => {
      const onChange = vi.fn();
      const start_time = new Date('2026-10-20T08:03:27.123Z');
      const end_time = new Date('2026-11-20T18:49:53.456Z');
      render(
        <Trigger
          onChange={onChange}
          initialProps={{
            catchup: false,
            maxConcurrentRuns: '4',
            trigger: { cron_schedule: { cron, start_time, end_time } },
          }}
        />,
      );
      expect(screen.getByLabelText('Repeat')).toHaveValue('custom');
      expect(screen.getByLabelText('Cron expression')).toHaveValue(cron);
      expect(onChange).toHaveBeenLastCalledWith({
        catchup: false,
        maxConcurrentRuns: '4',
        trigger: { cron_schedule: { cron, start_time, end_time } },
      });
      openAdvanced();
      change('Maximum concurrent runs', '5');
      expect(onChange).toHaveBeenLastCalledWith({
        catchup: false,
        maxConcurrentRuns: '5',
        trigger: { cron_schedule: { cron, start_time, end_time } },
      });
    },
  );

  it.each(['90', '3661', '7776000'])(
    'preserves existing exact interval %s seconds',
    (interval_second) => {
      const onChange = vi.fn();
      render(
        <Trigger
          onChange={onChange}
          initialProps={{ ...defaults, trigger: { periodic_schedule: { interval_second } } }}
        />,
      );
      expect(onChange).toHaveBeenLastCalledWith({
        ...defaults,
        trigger: {
          periodic_schedule: { ...periodic, interval_second },
        },
      });
      openAdvanced();
      change('Maximum concurrent runs', '2');
      expect(onChange).toHaveBeenLastCalledWith({
        ...defaults,
        maxConcurrentRuns: '2',
        trigger: {
          periodic_schedule: { ...periodic, interval_second },
        },
      });
    },
  );
});
