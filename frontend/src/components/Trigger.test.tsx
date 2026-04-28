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

const PARAMS_DEFAULT = {
  catchup: true,
  maxConcurrentRuns: '10',
};
const PERIODIC_DEFAULT = {
  end_time: undefined,
  interval_second: (60 * 60).toString(),
  start_time: undefined,
};
const CRON_DEFAULT = { cron: '0 0 * * * ?', end_time: undefined, start_time: undefined };

async function selectOption(selectElement: HTMLElement, optionText: string): Promise<void> {
  fireEvent.mouseDown(selectElement);
  const option = await screen.findByRole('option', { name: optionText });
  fireEvent.click(option);
}

function getTriggerTypeSelect(): HTMLElement {
  return screen.getByRole('combobox', { name: /trigger type/i });
}

function getIntervalCategorySelect(): HTMLElement {
  const comboboxes = screen.getAllByRole('combobox');
  return comboboxes[1];
}

describe('Trigger', () => {
  const RealDate = Date;

  function mockDate(isoDate: Date): void {
    (global as any).Date = class extends RealDate {
      constructor(...args: any[]) {
        super();
        if (args.length === 0) {
          return new RealDate(isoDate);
        }
        return new (RealDate as any)(...args);
      }
    };
  }

  beforeAll(() => {
    process.env.TZ = 'UTC';
    const now = new Date(2018, 11, 21, 7, 53);
    mockDate(now);
  });

  afterAll(() => {
    (global as any).Date = RealDate;
  });

  it('renders periodic schedule controls for initial render', () => {
    const { asFragment, unmount } = render(<Trigger />);
    expect(asFragment()).toMatchSnapshot();
    unmount();
  });

  it('renders periodic schedule controls if the trigger type is CRON', async () => {
    const { asFragment, unmount } = render(<Trigger />);
    await selectOption(getTriggerTypeSelect(), 'Cron');
    expect(asFragment()).toMatchSnapshot();
    unmount();
  });

  it('renders week days if the trigger type is CRON and interval is weekly', async () => {
    const { asFragment, unmount } = render(<Trigger />);
    await selectOption(getTriggerTypeSelect(), 'Cron');
    await selectOption(getIntervalCategorySelect(), 'Week');
    expect(asFragment()).toMatchSnapshot();
    unmount();
  });

  it('renders all week days enabled', async () => {
    const { asFragment, unmount } = render(<Trigger />);
    await selectOption(getTriggerTypeSelect(), 'Cron');
    await selectOption(getIntervalCategorySelect(), 'Week');
    fireEvent.click(screen.getByRole('checkbox', { name: 'All' }));
    expect(asFragment()).toMatchSnapshot();
    unmount();
  });

  it('enables a single day on click', async () => {
    const { asFragment, unmount } = render(<Trigger />);
    await selectOption(getTriggerTypeSelect(), 'Cron');
    await selectOption(getIntervalCategorySelect(), 'Week');
    fireEvent.click(screen.getByRole('button', { name: 'M' }));
    fireEvent.click(screen.getByRole('button', { name: 'W' }));
    expect(asFragment()).toMatchSnapshot();
    unmount();
  });

  describe('max concurrent run', () => {
    it('shows error message if the input is invalid (non-integer)', () => {
      render(<Trigger />);
      const maxConcurrencyParam = screen.getByDisplayValue('10');
      fireEvent.change(maxConcurrencyParam, { target: { value: '10a' } });
      screen.getByText('Invalid input. The maximum concurrent runs should be a positive integer.');
    });

    it('shows error message if the input is invalid (negative integer)', () => {
      render(<Trigger />);
      const maxConcurrencyParam = screen.getByDisplayValue('10');
      fireEvent.change(maxConcurrencyParam, { target: { value: '-10' } });
      screen.getByText('Invalid input. The maximum concurrent runs should be a positive integer.');
    });
  });

  describe('interval trigger', () => {
    it('builds an every-hour trigger by default', () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);

      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger with a start time if the checkbox is checked', () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      const now = new Date(2018, 11, 21, 7, 53);
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: { ...PERIODIC_DEFAULT, start_time: now },
        },
      });
    });

    it('builds trigger with the entered start date/time', () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      fireEvent.change(screen.getByLabelText('Start date'), {
        target: { value: '2018-11-23' },
      });
      fireEvent.change(screen.getByLabelText('Start time'), {
        target: { value: '08:35' },
      });
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
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      fireEvent.change(screen.getByLabelText('Start date'), {
        target: { value: '2018-11-23' },
      });
      fireEvent.change(screen.getByLabelText('Start time'), {
        target: { value: '' },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger without the entered start time if no date is entered', () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      fireEvent.change(screen.getByLabelText('Start date'), {
        target: { value: '' },
      });
      fireEvent.change(screen.getByLabelText('Start time'), {
        target: { value: '11:33' },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger with a date if both start and end checkboxes are checked', () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has end date' }));
      const now = new Date(2018, 11, 21, 7, 53);
      const oneWeekLater = new Date(2018, 11, 28, 7, 53);
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: { ...PERIODIC_DEFAULT, end_time: oneWeekLater, start_time: now },
        },
      });
    });

    it('resets trigger to no start date if it is added then removed', () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('Show invalid start date/time format message if date has wrong format.', async () => {
      render(<Trigger />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));

      // jsdom sanitizes type="date" inputs, rejecting non-YYYY-MM-DD values.
      // Temporarily switch to type="text" so the invalid value reaches handleChange.
      const startDateInput = screen.getByLabelText('Start date') as HTMLInputElement;
      startDateInput.type = 'text';
      fireEvent.change(startDateInput, {
        target: { value: 'this_is_not_valid_date_format' },
      });
      expect(
        await screen.findByText("Invalid start date or time, start time won't be set"),
      ).toBeInTheDocument();

      fireEvent.change(screen.getByLabelText('Start date'), {
        target: { value: '2021-01-01' },
      });
      expect(
        screen.queryByText("Invalid start date or time, start time won't be set"),
      ).not.toBeInTheDocument();
    });

    it('Hide invalid start date/time format message if start time checkbox is not selected.', async () => {
      render(<Trigger />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));

      const startDateInput = screen.getByLabelText('Start date') as HTMLInputElement;
      startDateInput.type = 'text';
      fireEvent.change(startDateInput, {
        target: { value: 'this_is_not_valid_date_format' },
      });
      expect(
        await screen.findByText("Invalid start date or time, start time won't be set"),
      ).toBeInTheDocument();

      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      expect(
        screen.queryByText("Invalid start date or time, start time won't be set"),
      ).not.toBeInTheDocument();
    });

    it('Show invalid end date/time format message if date has wrong format.', async () => {
      render(<Trigger />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has end date' }));

      const endTimeInput = screen.getByLabelText('End time') as HTMLInputElement;
      endTimeInput.type = 'text';
      fireEvent.change(endTimeInput, {
        target: { value: 'this_is_not_valid_time_format' },
      });
      expect(
        await screen.findByText("Invalid end date or time, end time won't be set"),
      ).toBeInTheDocument();

      fireEvent.change(screen.getByLabelText('End time'), {
        target: { value: '11:22' },
      });
      expect(
        screen.queryByText("Invalid end date or time, end time won't be set"),
      ).not.toBeInTheDocument();
    });

    it('Hide invalid end date/time format message if start time checkbox is not selected.', async () => {
      render(<Trigger />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has end date' }));

      const endTimeInput = screen.getByLabelText('End time') as HTMLInputElement;
      endTimeInput.type = 'text';
      fireEvent.change(endTimeInput, {
        target: { value: 'this_is_not_valid_date_format' },
      });
      expect(
        await screen.findByText("Invalid end date or time, end time won't be set"),
      ).toBeInTheDocument();

      fireEvent.click(screen.getByRole('checkbox', { name: 'Has end date' }));
      expect(
        screen.queryByText("Invalid end date or time, end time won't be set"),
      ).not.toBeInTheDocument();
    });

    it('builds trigger with a weekly interval', async () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      await selectOption(getIntervalCategorySelect(), 'Weeks');
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

    it('builds trigger with an every-three-months interval', async () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      await selectOption(getIntervalCategorySelect(), 'Months');
      fireEvent.change(screen.getByRole('spinbutton'), { target: { value: 3 } });
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
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      fireEvent.change(screen.getByDisplayValue('10'), { target: { value: '3' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        maxConcurrentRuns: '3',
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('builds trigger with the specified catchup setting', () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      fireEvent.click(screen.getByRole('checkbox', { name: 'Catchup' }));
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        catchup: false,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
    });

    it('inits with cloned initial props', () => {
      const spy = vi.fn();
      const startTime = new Date('2020-01-01T23:53:00.000Z');
      render(
        <Trigger
          onChange={spy}
          initialProps={{
            maxConcurrentRuns: '3',
            catchup: false,
            trigger: {
              periodic_schedule: {
                interval_second: '' + 60 * 60 * 3,
                start_time: startTime.toISOString() as any,
              },
            },
          }}
        />,
      );
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
    it('builds a 1-hour cron trigger by default', async () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      await selectOption(getTriggerTypeSelect(), 'Cron');
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: CRON_DEFAULT,
        },
      });
    });

    it('builds a 1-hour cron trigger with specified start date', async () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      await selectOption(getTriggerTypeSelect(), 'Cron');
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has start date' }));
      fireEvent.change(screen.getByLabelText('Start date'), {
        target: { value: '2018-03-23' },
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: {
            ...CRON_DEFAULT,
            start_time: new Date(2018, 2, 23, 7, 53),
            cron: '0 53 * * * ?',
          },
        },
      });
    });

    it('builds a daily cron trigger with specified end date/time', async () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      await selectOption(getTriggerTypeSelect(), 'Cron');
      fireEvent.click(screen.getByRole('checkbox', { name: 'Has end date' }));
      await selectOption(getIntervalCategorySelect(), 'Day');
      const oneWeekLater = new Date(2018, 11, 28, 7, 53);
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, end_time: oneWeekLater, cron: '0 0 0 * * ?' },
        },
      });
    });

    it('builds a weekly cron trigger that runs every Monday, Friday, and Saturday', async () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      await selectOption(getTriggerTypeSelect(), 'Cron');
      await selectOption(getIntervalCategorySelect(), 'Week');

      fireEvent.click(screen.getByRole('checkbox', { name: 'All' }));
      fireEvent.click(screen.getByRole('button', { name: 'M' }));
      fireEvent.click(screen.getByRole('button', { name: 'F' }));
      // Two "S" buttons exist (Sun, Sat); pick the second = Saturday
      fireEvent.click(screen.getAllByRole('button', { name: 'S' })[1]);
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, cron: '0 0 0 ? * 1,5,6' },
        },
      });
    });

    it('builds a cron with the manually specified cron string, even if days are toggled', async () => {
      const spy = vi.fn();
      render(<Trigger onChange={spy} />);
      await selectOption(getTriggerTypeSelect(), 'Cron');
      await selectOption(getIntervalCategorySelect(), 'Week');

      fireEvent.click(screen.getByRole('checkbox', { name: 'All' }));
      fireEvent.click(screen.getByRole('button', { name: 'M' }));
      fireEvent.click(screen.getByRole('button', { name: 'F' }));
      // Two "S" buttons exist (Sun, Sat); pick the second = Saturday
      fireEvent.click(screen.getAllByRole('button', { name: 'S' })[1]);

      fireEvent.click(screen.getByRole('checkbox', { name: /allow editing cron expression/i }));
      fireEvent.change(screen.getByLabelText('cron expression'), {
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
      const spy = vi.fn();
      const startTime = new Date('2020-01-01T00:00:00.000Z');
      const endTime = new Date('2020-01-02T01:02:00.000Z');
      render(
        <Trigger
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
