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
import { TextField } from './ui/text-field';
import { Button } from './ui/button';
import { Checkbox } from './ui/checkbox';
import { dateToPickerFormat, pickersToDate, TriggerSchedule } from '../lib/TriggerUtils';
import { scheduleDescription, describeCron } from '../lib/ScheduleDisplay';
import './RunFormFields.css';

type Preset = 'interval' | 'daily' | 'weekly' | 'monthly' | 'custom';
const units = { Second: 1, Minute: 60, Hour: 3600, Day: 86400, Week: 604800 };
type Unit = keyof typeof units;
interface Config {
  trigger?: TriggerSchedule;
  maxConcurrentRuns?: string;
  catchup: boolean;
}
interface Props {
  initialProps?: Partial<Config>;
  onChange?: (config: Config) => void;
}
interface Boundary {
  enabled: boolean;
  date: string;
  time: string;
  original?: Date;
}
interface State {
  preset: Preset;
  interval: string;
  unit: Unit;
  cron: string;
  originalCron?: string;
  runAt: string;
  days: number[];
  monthDay: string;
  start: Boundary;
  end: Boundary;
  concurrency: string;
  catchup: boolean;
}
function boundary(value: Date | undefined, fallback: Date): Boundary {
  const original = value ? new Date(value) : undefined;
  const [date, time] = dateToPickerFormat(original || fallback);
  return { enabled: !!value, date, time, original };
}
function boundaryValue(value: Boundary): Date | undefined {
  if (!value.enabled) return undefined;
  if (!value.date || !value.time) throw new Error('Choose both a date and time.');
  // Untouched clones retain seconds/milliseconds and their original instant across DST folds.
  if (value.original) {
    const [date, time] = dateToPickerFormat(value.original);
    if (date === value.date && time === value.time) return value.original;
  }
  return pickersToDate(true, value.date, value.time);
}
function schedule(state: State): { trigger?: TriggerSchedule; error?: string } {
  try {
    const start_time = boundaryValue(state.start);
    const end_time = boundaryValue(state.end);
    if (start_time && end_time && end_time <= start_time)
      throw new Error('End must be after start.');
    if (state.preset === 'interval') {
      const count = Number(state.interval);
      const seconds = count * units[state.unit];
      if (!Number.isSafeInteger(seconds) || seconds < 1 || count <= 0)
        throw new Error('Enter a positive interval in whole seconds.');
      return {
        trigger: { periodic_schedule: { interval_second: String(seconds), start_time, end_time } },
      };
    }
    let cron = state.cron;
    if (state.preset !== 'custom') {
      if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(state.runAt))
        throw new Error('Choose a valid run time.');
      const [hour, minute] = state.runAt.split(':').map(Number);
      if (state.preset === 'weekly' && !state.days.length)
        throw new Error('Choose at least one day.');
      if (
        state.preset === 'monthly' &&
        (!/^\d+$/.test(state.monthDay) || +state.monthDay < 1 || +state.monthDay > 31)
      )
        throw new Error('Choose a day from 1 to 31.');
      cron = `0 ${minute} ${hour} ${state.preset === 'monthly' ? state.monthDay : '*'} * ${state.preset === 'weekly' ? state.days.join(',') : '*'}`;
    }
    if (cron !== state.originalCron) describeCron(cron);
    return { trigger: { cron_schedule: { cron, start_time, end_time } } };
  } catch (error) {
    return { error: error instanceof Error ? error.message : 'Enter a valid cron expression.' };
  }
}

export default class Trigger extends React.Component<Props, State> {
  public state: State = (() => {
    const initial = this.props.initialProps;
    const trigger = initial?.trigger;
    const source = trigger?.cron_schedule || trigger?.periodic_schedule;
    const seconds = Number(trigger?.periodic_schedule?.interval_second || 3600);
    const unit =
      (['Week', 'Day', 'Hour', 'Minute', 'Second'] as Unit[]).find(
        (unit) => seconds > 0 && seconds % units[unit] === 0,
      ) || 'Second';
    const today = new Date();
    return {
      preset: trigger?.cron_schedule ? 'custom' : 'interval',
      interval: String(seconds / units[unit]),
      unit,
      cron: trigger?.cron_schedule?.cron || '',
      originalCron: trigger?.cron_schedule?.cron,
      runAt: '09:00',
      days: [1, 2, 3, 4, 5],
      monthDay: '1',
      start: boundary(source?.start_time, today),
      end: boundary(source?.end_time, new Date(today.getTime() + 7 * 86400000)),
      concurrency: initial?.maxConcurrentRuns ?? '10',
      catchup: initial?.catchup ?? true,
    };
  })();
  public componentDidMount(): void {
    this.publish();
  }
  private publish = () => {
    this.props.onChange?.({
      trigger: schedule(this.state).trigger,
      maxConcurrentRuns:
        Number.isInteger(+this.state.concurrency) &&
        +this.state.concurrency >= 1 &&
        +this.state.concurrency <= 10
          ? this.state.concurrency
          : undefined,
      catchup: this.state.catchup,
    });
  };
  private change = (patch: Partial<State>) =>
    this.setState(patch as Pick<State, keyof State>, this.publish);
  private renderBoundary = (key: 'start' | 'end') => {
    const value = this.state[key];
    const label = key === 'start' ? 'Start' : 'End';
    const update = (patch: Partial<Boundary>) => this.change({ [key]: { ...value, ...patch } });
    return (
      <div className='kfp-schedule-window'>
        <label className='kfp-native-field'>
          <span>{label}s</span>
          <select
            aria-label={`${label}s`}
            value={value.enabled ? 'date' : key === 'start' ? 'now' : 'never'}
            onChange={(e) => update({ enabled: e.target.value === 'date' })}
          >
            <option value={key === 'start' ? 'now' : 'never'}>
              {key === 'start' ? 'Start when enabled' : 'No end date'}
            </option>
            <option value='date'>Choose date</option>
          </select>
        </label>
        {value.enabled && (
          <div className='kfp-schedule-date-time'>
            <TextField
              label={`${label} date`}
              error={!value.date || !value.time ? 'Choose both a date and time.' : undefined}
              type='date'
              value={value.date}
              onChange={(e) => update({ date: e.target.value, original: undefined })}
            />
            <TextField
              label={`${label} time`}
              error={!value.date || !value.time ? 'Choose both a date and time.' : undefined}
              type='time'
              value={value.time}
              onChange={(e) => update({ time: e.target.value, original: undefined })}
            />
          </div>
        )}
      </div>
    );
  };
  public render(): React.JSX.Element {
    const s = this.state;
    const result = schedule(s);
    const concurrencyValid =
      Number.isInteger(Number(s.concurrency)) && +s.concurrency >= 1 && +s.concurrency <= 10;
    const localZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    return (
      <fieldset className='kfp-run-form-fields kfp-trigger'>
        <legend>Schedule</legend>
        <div className='kfp-schedule-frequency'>
          <label className='kfp-native-field'>
            <span>Repeat</span>
            <select
              aria-label='Repeat'
              value={s.preset}
              onChange={(e) => this.change({ preset: e.target.value as Preset })}
            >
              <option value='interval'>Every interval</option>
              <option value='daily'>Daily</option>
              <option value='weekly'>Weekly</option>
              <option value='monthly'>Monthly</option>
              <option value='custom'>Custom cron</option>
            </select>
          </label>
          {s.preset === 'interval' && (
            <div className='kfp-schedule-interval'>
              <TextField
                label='Interval'
                type='number'
                min={1}
                value={s.interval}
                onChange={(e) => this.change({ interval: e.target.value })}
              />
              <label className='kfp-native-field'>
                <span>Interval unit</span>
                <select
                  aria-label='Interval unit'
                  value={s.unit}
                  onChange={(e) => this.change({ unit: e.target.value as Unit })}
                >
                  {Object.keys(units).map((unit) => (
                    <option key={unit} value={unit}>
                      {unit.toLowerCase()}
                      {Number(s.interval) === 1 ? '' : 's'}
                    </option>
                  ))}
                </select>
              </label>
            </div>
          )}
          {['daily', 'weekly', 'monthly'].includes(s.preset) && (
            <TextField
              label='Run at'
              type='time'
              value={s.runAt}
              onChange={(e) => this.change({ runAt: e.target.value })}
            />
          )}
          {s.preset === 'monthly' && (
            <TextField
              label='Day of month'
              type='number'
              min={1}
              max={31}
              value={s.monthDay}
              onChange={(e) => this.change({ monthDay: e.target.value })}
            />
          )}
        </div>
        {s.preset === 'weekly' && (
          <div className='kfp-schedule-weekdays' role='group' aria-label='Days of week'>
            {['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'].map(
              (day, index) => (
                <Button
                  key={day}
                  variant={s.days.includes(index) ? 'default' : 'secondary'}
                  aria-label={day}
                  aria-pressed={s.days.includes(index)}
                  onClick={() =>
                    this.change({
                      days: s.days.includes(index)
                        ? s.days.filter((d) => d !== index)
                        : [...s.days, index].sort(),
                    })
                  }
                >
                  {day.slice(0, 3)}
                </Button>
              ),
            )}
          </div>
        )}
        {s.preset === 'custom' && (
          <div>
            <TextField
              label='Cron expression'
              value={s.cron}
              onChange={(e) => this.change({ cron: e.target.value })}
              error={result.error}
            />
            <p className='kfp-form-help'>
              Seconds · minutes · hours · day of month · month · day of week. Example: 0 30 9 * *
              MON-FRI.
            </p>
          </div>
        )}
        <div className='kfp-schedule-summary' aria-live='polite'>
          {result.error ? (
            <p role='alert' className='kfp-form-error'>
              {result.error}
            </p>
          ) : (
            <strong>{scheduleDescription(result.trigger)}</strong>
          )}
          <p>
            {s.preset === 'interval'
              ? 'Fixed time between scheduled starts, not between completed runs.'
              : 'Calendar schedules use the cluster scheduler’s timezone (UTC in the default installation). The cluster does not expose its configured timezone here.'}
          </p>
          {s.preset === 'monthly' && +s.monthDay > 28 && (
            <p>Months without day {s.monthDay} are skipped.</p>
          )}
        </div>
        <div className='kfp-schedule-windows'>
          {this.renderBoundary('start')}
          {this.renderBoundary('end')}
        </div>
        {(s.start.enabled || s.end.enabled) && (
          <p className='kfp-form-help'>
            Start and end dates use your local timezone: {localZone}. They bound the schedule; they
            do not change its repeat time.
          </p>
        )}
        <details className='kfp-schedule-advanced'>
          <summary>Advanced execution settings</summary>
          <div className='kfp-run-form-fields'>
            <TextField
              label='Maximum concurrent runs'
              inputMode='numeric'
              value={s.concurrency}
              onChange={(e) => this.change({ concurrency: e.target.value })}
              error={concurrencyValid ? undefined : 'Enter a whole number from 1 to 10.'}
            />
            <label className='kfp-form-check'>
              <Checkbox
                checked={s.catchup}
                onCheckedChange={(catchup) => this.change({ catchup })}
              />
              Run missed schedules when resumed
            </label>
            <p className='kfp-form-help'>
              When enabled, missed schedule times are backfilled after a pause or delay. This can
              create many runs. Turn it off if the pipeline handles backfill itself.
            </p>
          </div>
        </details>
      </fieldset>
    );
  }
}
