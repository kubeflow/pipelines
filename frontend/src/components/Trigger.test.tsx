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
import { act, fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import Trigger from './Trigger';
import { TriggerType, PeriodicInterval } from '../lib/TriggerUtils';

type TriggerState = Trigger['state'];

type TriggerProps = React.ComponentProps<typeof Trigger>;

class TriggerWrapper {
  private _instance: Trigger;
  private _renderResult: ReturnType<typeof render>;

  public constructor(instance: Trigger, renderResult: ReturnType<typeof render>) {
    this._instance = instance;
    this._renderResult = renderResult;
  }

  public instance(): Trigger {
    return this._instance;
  }

  public state<K extends keyof TriggerState>(key?: K): TriggerState | TriggerState[K] {
    const state = this._instance.state;
    return key ? state[key] : state;
  }

  public renderResult(): ReturnType<typeof render> {
    return this._renderResult;
  }

  public unmount(): void {
    this._renderResult.unmount();
  }
}

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

function renderTrigger(props: TriggerProps = {}): TriggerWrapper {
  const ref = React.createRef<Trigger>();
  const renderResult = render(<Trigger ref={ref} {...props} />);
  if (!ref.current) {
    throw new Error('Trigger instance not available');
  }
  return new TriggerWrapper(ref.current, renderResult);
}

async function applyChange(
  wrapper: TriggerWrapper,
  name: string,
  event: { target: { value?: any; type?: string; checked?: boolean } },
): Promise<void> {
  await act(async () => {
    wrapper.instance().handleChange(name)(event);
  });
}

describe('Trigger', () => {
  // tslint:disable-next-line:variable-name
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
    const wrapper = renderTrigger();
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders periodic schedule controls if the trigger type is CRON', async () => {
    const wrapper = renderTrigger();
    await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders week days if the trigger type is CRON and interval is weekly', async () => {
    const wrapper = renderTrigger();
    await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
    await applyChange(wrapper, 'intervalCategory', { target: { value: PeriodicInterval.WEEK } });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('renders all week days enabled', async () => {
    const wrapper = renderTrigger();
    await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
    await applyChange(wrapper, 'intervalCategory', { target: { value: PeriodicInterval.WEEK } });
    await act(async () => {
      (wrapper.instance() as any)._toggleCheckAllDays();
    });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  it('enables a single day on click', async () => {
    const wrapper = renderTrigger();
    await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
    await applyChange(wrapper, 'intervalCategory', { target: { value: PeriodicInterval.WEEK } });
    await act(async () => {
      (wrapper.instance() as any)._toggleDay(1);
      (wrapper.instance() as any)._toggleDay(3);
    });
    expect(wrapper.renderResult().asFragment()).toMatchSnapshot();
    wrapper.unmount();
  });

  describe('max concurrent run', () => {
    it('shows error message if the input is invalid (non-integer)', () => {
      render(<Trigger />);
      const maxConcurrenyParam = screen.getByDisplayValue('10');
      fireEvent.change(maxConcurrenyParam, { target: { value: '10a' } });
      screen.getByText('Invalid input. The maximum concurrent runs should be a positive integer.');
    });

    it('shows error message if the input is invalid (negative integer)', () => {
      render(<Trigger />);
      const maxConcurrenyParam = screen.getByDisplayValue('10');
      fireEvent.change(maxConcurrenyParam, { target: { value: '-10' } });
      screen.getByText('Invalid input. The maximum concurrent runs should be a positive integer.');
    });
  });

  describe('interval trigger', () => {
    it('builds an every-hour trigger by default', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
      wrapper.unmount();
    });

    it('builds trigger with a start time if the checkbox is checked', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });
      const now = new Date(2018, 11, 21, 7, 53);
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: { ...PERIODIC_DEFAULT, start_time: now },
        },
      });
      wrapper.unmount();
    });

    it('builds trigger with the entered start date/time', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });
      await applyChange(wrapper, 'startDate', { target: { value: '2018-11-23' } });
      await applyChange(wrapper, 'startTime', { target: { value: '08:35' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: {
            ...PERIODIC_DEFAULT,
            start_time: new Date(2018, 10, 23, 8, 35),
          },
        },
      });
      wrapper.unmount();
    });

    it('builds trigger without the entered start date if no time is entered', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });
      await applyChange(wrapper, 'startDate', { target: { value: '2018-11-23' } });
      await applyChange(wrapper, 'startTime', { target: { value: '' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
      wrapper.unmount();
    });

    it('builds trigger without the entered start time if no date is entered', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });
      await applyChange(wrapper, 'startDate', { target: { value: '' } });
      await applyChange(wrapper, 'startTime', { target: { value: '11:33' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
      wrapper.unmount();
    });

    it('builds trigger with a date if both start and end checkboxes are checked', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });
      await applyChange(wrapper, 'hasEndDate', { target: { type: 'checkbox', checked: true } });
      const now = new Date(2018, 11, 21, 7, 53);
      const oneWeekLater = new Date(2018, 11, 28, 7, 53);
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: { ...PERIODIC_DEFAULT, end_time: oneWeekLater, start_time: now },
        },
      });
      wrapper.unmount();
    });

    it('resets trigger to no start date if it is added then removed', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: false } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
      wrapper.unmount();
    });

    it('Show invalid start date/time format message if date has wrong format.', async () => {
      const wrapper = renderTrigger();
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });

      await applyChange(wrapper, 'startDate', {
        target: { value: 'this_is_not_valid_date_format' },
      });
      let messageBox = screen.getByTestId('startTimeMessage');
      expect(messageBox.textContent).toEqual("Invalid start date or time, start time won't be set");

      await applyChange(wrapper, 'startDate', { target: { value: '2021-01-01' } });
      messageBox = screen.getByTestId('startTimeMessage');
      expect(messageBox.textContent).toEqual('');
      wrapper.unmount();
    });

    it('Hide invalid start date/time format message if start time checkbox is not selected.', async () => {
      const wrapper = renderTrigger();
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });

      await applyChange(wrapper, 'startDate', {
        target: { value: 'this_is_not_valid_date_format' },
      });
      let messageBox = screen.getByTestId('startTimeMessage');
      expect(messageBox.textContent).toEqual("Invalid start date or time, start time won't be set");

      await applyChange(wrapper, 'hasStartDate', {
        target: { type: 'checkbox', checked: false },
      });
      messageBox = screen.getByTestId('startTimeMessage');
      expect(messageBox.textContent).toEqual('');
      wrapper.unmount();
    });

    it('Show invalid end date/time format message if date has wrong format.', async () => {
      const wrapper = renderTrigger();
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasEndDate', { target: { type: 'checkbox', checked: true } });

      await applyChange(wrapper, 'endTime', { target: { value: 'this_is_not_valid_time_format' } });
      let messageBox = screen.getByTestId('endTimeMessage');
      expect(messageBox.textContent).toEqual("Invalid end date or time, end time won't be set");

      await applyChange(wrapper, 'endTime', { target: { value: '11:22' } });
      messageBox = screen.getByTestId('endTimeMessage');
      expect(messageBox.textContent).toEqual('');
      wrapper.unmount();
    });

    it('Hide invalid end date/time format message if start time checkbox is not selected.', async () => {
      const wrapper = renderTrigger();
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'hasEndDate', { target: { type: 'checkbox', checked: true } });

      await applyChange(wrapper, 'endTime', {
        target: { value: 'this_is_not_valid_date_format' },
      });
      let messageBox = screen.getByTestId('endTimeMessage');
      expect(messageBox.textContent).toEqual("Invalid end date or time, end time won't be set");

      await applyChange(wrapper, 'hasEndDate', { target: { type: 'checkbox', checked: false } });
      messageBox = screen.getByTestId('endTimeMessage');
      expect(messageBox.textContent).toEqual('');
      wrapper.unmount();
    });

    it('builds trigger with a weekly interval', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'intervalCategory', { target: { value: PeriodicInterval.WEEK } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: {
            ...PERIODIC_DEFAULT,
            interval_second: (7 * 24 * 60 * 60).toString(),
          },
        },
      });
      wrapper.unmount();
    });

    it('builds trigger with an every-three-months interval', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'intervalCategory', { target: { value: PeriodicInterval.MONTH } });
      await applyChange(wrapper, 'intervalValue', { target: { value: 3 } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          periodic_schedule: {
            ...PERIODIC_DEFAULT,
            interval_second: (3 * 30 * 24 * 60 * 60).toString(),
          },
        },
      });
      wrapper.unmount();
    });

    it('builds trigger with the specified max concurrency setting', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'maxConcurrentRuns', { target: { value: '3' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        maxConcurrentRuns: '3',
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
      wrapper.unmount();
    });

    it('builds trigger with the specified catchup setting', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.INTERVALED } });
      await applyChange(wrapper, 'catchup', { target: { type: 'checkbox', checked: false } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        catchup: false,
        trigger: {
          periodic_schedule: PERIODIC_DEFAULT,
        },
      });
      wrapper.unmount();
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
    it('builds a 1-hour cron trigger by default', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: CRON_DEFAULT,
        },
      });
      wrapper.unmount();
    });

    it('builds a 1-hour cron trigger with specified start date', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
      await applyChange(wrapper, 'hasStartDate', { target: { type: 'checkbox', checked: true } });
      await applyChange(wrapper, 'startDate', { target: { value: '2018-03-23' } });
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
      wrapper.unmount();
    });

    it('builds a daily cron trigger with specified end date/time', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
      await applyChange(wrapper, 'hasEndDate', { target: { type: 'checkbox', checked: true } });
      await applyChange(wrapper, 'intervalCategory', { target: { value: PeriodicInterval.DAY } });
      const oneWeekLater = new Date(2018, 11, 28, 7, 53);
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, end_time: oneWeekLater, cron: '0 0 0 * * ?' },
        },
      });
      wrapper.unmount();
    });

    it('builds a weekly cron trigger that runs every Monday, Friday, and Saturday', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
      await applyChange(wrapper, 'intervalCategory', { target: { value: PeriodicInterval.WEEK } });
      await act(async () => {
        (wrapper.instance() as any)._toggleCheckAllDays();
        (wrapper.instance() as any)._toggleDay(1);
        (wrapper.instance() as any)._toggleDay(5);
        (wrapper.instance() as any)._toggleDay(6);
      });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, cron: '0 0 0 ? * 1,5,6' },
        },
      });
      wrapper.unmount();
    });

    it('builds a cron with the manually specified cron string, even if days are toggled', async () => {
      const spy = vi.fn();
      const wrapper = renderTrigger({ onChange: spy });
      await applyChange(wrapper, 'type', { target: { value: TriggerType.CRON } });
      await applyChange(wrapper, 'intervalCategory', { target: { value: PeriodicInterval.WEEK } });
      await act(async () => {
        (wrapper.instance() as any)._toggleCheckAllDays();
        (wrapper.instance() as any)._toggleDay(1);
        (wrapper.instance() as any)._toggleDay(5);
        (wrapper.instance() as any)._toggleDay(6);
      });
      await applyChange(wrapper, 'editCron', { target: { type: 'checkbox', checked: true } });
      await applyChange(wrapper, 'cron', { target: { value: 'oops this will break!' } });
      expect(spy).toHaveBeenLastCalledWith({
        ...PARAMS_DEFAULT,
        trigger: {
          cron_schedule: { ...CRON_DEFAULT, cron: 'oops this will break!' },
        },
      });
      wrapper.unmount();
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
