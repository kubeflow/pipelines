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
import { ExternalLink } from '../atoms/ExternalLink';
import { TextField } from './ui/text-field';
import { Button } from './ui/button';
import { Checkbox } from './ui/checkbox';
import './RunFormFields.css';
import {
  buildCron,
  buildTrigger,
  dateToPickerFormat,
  PeriodicInterval,
  TriggerSchedule,
  pickersToDate,
  triggers,
  TriggerType,
  parseTrigger,
  ParsedTrigger,
} from '../lib/TriggerUtils';
import { logger } from 'src/lib/Utils';

type TriggerInitialProps = {
  maxConcurrentRuns?: string;
  catchup?: boolean;
  trigger?: TriggerSchedule;
};

interface TriggerProps {
  initialProps?: TriggerInitialProps;
  onChange?: (config: {
    trigger?: TriggerSchedule;
    maxConcurrentRuns?: string;
    catchup: boolean;
  }) => void;
}

interface TriggerState {
  cron: string;
  editCron: boolean;
  endDate: string;
  endTime: string;
  hasEndDate: boolean;
  hasStartDate: boolean;
  intervalCategory: PeriodicInterval;
  intervalValue: number;
  maxConcurrentRuns: string;
  selectedDays: boolean[];
  startDate: string;
  startTime: string;
  type: TriggerType;
  catchup: boolean;
  startTimeMessage: string;
  endTimeMessage: string;
}

function ScheduleDateTimeFields({
  boundary,
  date,
  time,
  visible,
  message,
  onDateChange,
  onTimeChange,
}: {
  boundary: 'start' | 'end';
  date: string;
  time: string;
  visible: boolean;
  message: string;
  onDateChange: React.ChangeEventHandler<HTMLInputElement>;
  onTimeChange: React.ChangeEventHandler<HTMLInputElement>;
}) {
  const messageId = React.useId();
  const label = boundary === 'start' ? 'Start' : 'End';
  return (
    <>
      <div
        className='kfp-schedule-date-time'
        style={{ visibility: visible ? 'visible' : 'hidden' }}
      >
        <TextField
          label={`${label} date`}
          type='date'
          onChange={onDateChange}
          value={date}
          aria-invalid={!!message}
          aria-describedby={message ? messageId : undefined}
        />
        <TextField
          label={`${label} time`}
          type='time'
          onChange={onTimeChange}
          value={time}
          aria-invalid={!!message}
          aria-describedby={message ? messageId : undefined}
        />
      </div>
      <div
        id={messageId}
        data-testid={`${boundary}TimeMessage`}
        className='kfp-form-error'
        role={message ? 'alert' : undefined}
        style={{ visibility: visible ? 'visible' : 'hidden' }}
      >
        {message}
      </div>
    </>
  );
}

export default class Trigger extends React.Component<TriggerProps, TriggerState> {
  public state: TriggerState = (() => {
    const { maxConcurrentRuns, catchup, trigger } =
      this.props.initialProps || ({} as TriggerInitialProps);
    let parsedTrigger: Partial<ParsedTrigger> = {};
    try {
      if (trigger) {
        parsedTrigger = parseTrigger(trigger);
      }
    } catch (err) {
      logger.warn('Failed to parse original trigger: ', trigger);
      logger.warn(err);
    }
    const startDateTime = parsedTrigger.startDateTime ?? new Date();
    const endDateTime =
      parsedTrigger.endDateTime ??
      new Date(
        startDateTime.getFullYear(),
        startDateTime.getMonth(),
        startDateTime.getDate() + 7,
        startDateTime.getHours(),
        startDateTime.getMinutes(),
      );
    const [startDate, startTime] = dateToPickerFormat(startDateTime);
    const [endDate, endTime] = dateToPickerFormat(endDateTime);

    return {
      catchup: catchup ?? true,
      maxConcurrentRuns: maxConcurrentRuns || '10',
      hasEndDate: !!parsedTrigger?.endDateTime,
      endDate,
      endTime,
      hasStartDate: !!parsedTrigger?.startDateTime,
      startDate,
      startTime,
      selectedDays: new Array(7).fill(true),
      type: parsedTrigger.type ?? TriggerType.INTERVALED,
      // cron state
      editCron: parsedTrigger.type === TriggerType.CRON,
      cron: parsedTrigger.cron || '',
      // interval state
      intervalCategory: parsedTrigger.intervalCategory ?? PeriodicInterval.HOUR,
      intervalValue: parsedTrigger.intervalValue ?? 1,
      startTimeMessage: '',
      endTimeMessage: '',
    };
  })();

  public componentDidMount(): void {
    // TODO: This is called here because NewRun only updates its Trigger in state when onChange is
    // called on the Trigger, which without this may never happen if a user doesn't interact with
    // the Trigger. NewRun should probably keep the Trigger state and pass it down as a prop to this
    this._updateTrigger();
  }

  public render(): React.JSX.Element {
    const {
      cron,
      editCron,
      endDate,
      endTime,
      hasEndDate,
      hasStartDate,
      intervalCategory,
      intervalValue,
      maxConcurrentRuns,
      selectedDays,
      startDate,
      startTime,
      type,
      catchup,
      startTimeMessage,
      endTimeMessage,
    } = this.state;

    return (
      <fieldset className='kfp-run-form-fields kfp-trigger'>
        <legend>Schedule timing</legend>
        <label className='kfp-native-field'>
          <span>
            Trigger type<span aria-hidden='true'> *</span>
          </span>
          <select
            aria-label='Trigger type'
            value={type}
            required
            onChange={(event) =>
              this.setState(
                { type: Number(event.target.value) as TriggerType },
                this._updateTrigger,
              )
            }
          >
            {Array.from(triggers.entries()).map(([value, trigger]) => (
              <option key={value} value={value}>
                {trigger.displayName}
              </option>
            ))}
          </select>
        </label>
        <TextField
          label='Maximum concurrent runs'
          required
          inputMode='numeric'
          onChange={this.handleChange('maxConcurrentRuns')}
          value={maxConcurrentRuns}
          error={
            Number.isInteger(Number(maxConcurrentRuns)) && Number(maxConcurrentRuns) > 0
              ? undefined
              : 'Invalid input. The maximum concurrent runs should be a positive integer.'
          }
        />

        <fieldset className='kfp-schedule-boundary'>
          <legend>Start</legend>
          <label className='kfp-form-check'>
            <Checkbox
              checked={hasStartDate}
              onCheckedChange={this._checkedChanged('hasStartDate')}
            />
            Has start date
          </label>
          <ScheduleDateTimeFields
            boundary='start'
            date={startDate}
            time={startTime}
            visible={hasStartDate}
            message={startTimeMessage}
            onDateChange={this.handleChange('startDate')}
            onTimeChange={this.handleChange('startTime')}
          />
        </fieldset>
        <fieldset className='kfp-schedule-boundary'>
          <legend>End</legend>
          <label className='kfp-form-check'>
            <Checkbox checked={hasEndDate} onCheckedChange={this._checkedChanged('hasEndDate')} />
            Has end date
          </label>
          <ScheduleDateTimeFields
            boundary='end'
            date={endDate}
            time={endTime}
            visible={hasEndDate}
            message={endTimeMessage}
            onDateChange={this.handleChange('endDate')}
            onTimeChange={this.handleChange('endTime')}
          />
        </fieldset>
        <div>
          <label className='kfp-form-check'>
            <Checkbox checked={catchup} onCheckedChange={this._checkedChanged('catchup')} />
            Catchup
          </label>
          <details className='kfp-form-help'>
            <summary>About catchup</summary>
            <p>Whether the recurring run should catch up if behind schedule. Defaults to true.</p>
            <p>
              For example, if the recurring run is paused for a while and re-enabled afterwards. If
              catchup=true, the scheduler will catch up on (backfill) each missed interval.
              Otherwise, it only schedules the latest interval if more than one interval is ready to
              be scheduled.
            </p>
            <p>
              Usually, if your pipeline handles backfill internally, you should turn catchup off to
              avoid duplicate backfill.
            </p>
          </details>
        </div>
        <fieldset className='kfp-schedule-boundary'>
          <legend>Run every</legend>
          <div className='kfp-schedule-interval'>
            {type === TriggerType.INTERVALED && (
              <TextField
                label='Interval'
                required
                type='number'
                min={1}
                onChange={this.handleChange('intervalValue')}
                value={intervalValue}
                error={intervalValue < 1 ? 'Interval must be at least 1.' : undefined}
              />
            )}
            <label className='kfp-native-field'>
              <span>Interval unit</span>
              <select
                aria-label='Interval unit'
                required
                onChange={this.handleChange('intervalCategory')}
                value={intervalCategory}
              >
                {Object.values(PeriodicInterval).map((interval) => (
                  <option key={interval} value={interval}>
                    {interval + (type === TriggerType.INTERVALED ? 's' : '')}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </fieldset>
        {type === TriggerType.CRON && (
          <>
            {intervalCategory === PeriodicInterval.WEEK && (
              <fieldset className='kfp-schedule-boundary'>
                <legend>On</legend>
                <label className='kfp-form-check'>
                  <Checkbox
                    checked={this._isAllDaysChecked()}
                    onCheckedChange={this._toggleCheckAllDays.bind(this)}
                  />
                  All
                </label>
                <div className='kfp-schedule-weekdays'>
                  {[
                    'Sunday',
                    'Monday',
                    'Tuesday',
                    'Wednesday',
                    'Thursday',
                    'Friday',
                    'Saturday',
                  ].map((day, i) => (
                    <Button
                      key={day}
                      size='icon'
                      variant={selectedDays[i] ? 'default' : 'secondary'}
                      aria-label={day}
                      aria-pressed={selectedDays[i]}
                      onClick={() => this._toggleDay(i)}
                    >
                      {day[0]}
                    </Button>
                  ))}
                </div>
              </fieldset>
            )}
            <div>
              <label className='kfp-form-check'>
                <Checkbox checked={editCron} onCheckedChange={this._checkedChanged('editCron')} />
                Allow editing cron expression.
              </label>
              <p className='kfp-form-help'>
                Cron expression format is specified
                <ExternalLink href='https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format'>
                  {' here'}
                </ExternalLink>
                .
              </p>
              <TextField
                label='cron expression'
                onChange={this.handleChange('cron')}
                value={cron}
                disabled={!editCron}
              />
              <p className='kfp-form-help'>
                Note: Start and end dates/times are handled outside of cron.
              </p>
            </div>
          </>
        )}
      </fieldset>
    );
  }

  private _checkedChanged =
    (name: 'hasStartDate' | 'hasEndDate' | 'catchup' | 'editCron') => (checked: boolean) => {
      this.setState((state) => ({ ...state, [name]: checked }), this._updateTrigger);
    };

  public handleChange = (name: string) => (event: any) => {
    const target = event.target;
    const value = target.type === 'checkbox' ? target.checked : target.value;
    // Make sure the desired field is set on the state object first, then
    // use the state values to compute the new trigger
    this.setState(
      {
        [name]: value,
      } as any,
      this._updateTrigger,
    );
  };

  private _updateTrigger = () => {
    const {
      hasStartDate,
      hasEndDate,
      startDate,
      startTime,
      endDate,
      endTime,
      editCron,
      intervalCategory,
      intervalValue,
      type,
      cron,
      selectedDays,
      catchup,
    } = this.state;

    var startDateTime: Date | undefined = undefined;
    var endDateTime: Date | undefined = undefined;
    var startTimeMessage = '';
    var endTimeMessage = '';

    try {
      if (hasStartDate) {
        startDateTime = pickersToDate(hasStartDate, startDate, startTime);
      }
    } catch (e) {
      if (e instanceof Error && e.message === 'Invalid picker format') {
        startTimeMessage = "Invalid start date or time, start time won't be set";
      } else {
        throw e;
      }
    }

    try {
      if (hasEndDate) {
        endDateTime = pickersToDate(hasEndDate, endDate, endTime);
      }
    } catch (e) {
      if (e instanceof Error && e.message === 'Invalid picker format') {
        endTimeMessage = "Invalid end date or time, end time won't be set";
      } else {
        throw e;
      }
    }

    this.setState({
      startTimeMessage: startTimeMessage,
      endTimeMessage: endTimeMessage,
    });

    // TODO: Why build the cron string unless the TriggerType is not CRON?
    // Unless cron editing is enabled, calculate the new cron string, set it in state,
    // then use it to build new trigger object and notify the parent
    this.setState(
      {
        cron: editCron ? cron : buildCron(startDateTime, intervalCategory, selectedDays),
      },
      () => {
        const trigger = buildTrigger(
          intervalCategory,
          intervalValue,
          startDateTime,
          endDateTime,
          type,
          this.state.cron,
        );

        if (this.props.onChange) {
          this.props.onChange({
            catchup,
            maxConcurrentRuns: trigger ? this.state.maxConcurrentRuns : undefined,
            trigger,
          });
        }
      },
    );
  };

  private _isAllDaysChecked(): boolean {
    return this.state.selectedDays.every((d) => !!d);
  }

  private _toggleCheckAllDays(): void {
    const isAllChecked = this._isAllDaysChecked();
    this.state.selectedDays.forEach((d, i) => {
      if (d !== !isAllChecked) {
        this._toggleDay(i);
      }
    });
  }

  private _toggleDay(index: number): void {
    const newDays = this.state.selectedDays;
    newDays[index] = !newDays[index];
    const startDate = pickersToDate(
      this.state.hasStartDate,
      this.state.startDate,
      this.state.startTime,
    );
    const cron = buildCron(startDate, this.state.intervalCategory, this.state.selectedDays);

    this.setState(
      {
        cron,
        selectedDays: newDays,
      },
      this._updateTrigger,
    );
  }
}
