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

import Button from '@material-ui/core/Button';
import Checkbox from '@material-ui/core/Checkbox';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import MenuItem from '@material-ui/core/MenuItem';
import * as React from 'react';
import { stylesheet } from 'typestyle';
import { ApiTrigger } from '../apis/job';
import { HelpButton } from '../atoms/HelpButton';
import Input from '../atoms/Input';
import Separator from '../atoms/Separator';
import { commonCss } from '../Css';
import {
  buildCron,
  buildTrigger,
  dateToPickerFormat,
  PeriodicInterval,
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
  trigger?: ApiTrigger;
};

interface TriggerProps {
  initialProps?: TriggerInitialProps;
  onChange?: (config: {
    trigger?: ApiTrigger;
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
}

const css = stylesheet({
  noMargin: {
    margin: 0,
  },
});

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
      intervalCategory: parsedTrigger.intervalCategory ?? PeriodicInterval.MINUTE,
      intervalValue: parsedTrigger.intervalValue ?? 1,
    };
  })();

  public componentDidMount(): void {
    // TODO: This is called here because NewRun only updates its Trigger in state when onChange is
    // called on the Trigger, which without this may never happen if a user doesn't interact with
    // the Trigger. NewRun should probably keep the Trigger state and pass it down as a prop to this
    this._updateTrigger();
  }

  public render(): JSX.Element {
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
    } = this.state;

    return (
      <div>
        <Input
          select={true}
          label='Trigger type'
          required={true}
          onChange={this.handleChange('type')}
          value={type}
          variant='outlined'
        >
          {Array.from(triggers.entries()).map((trigger, i) => (
            <MenuItem key={i} value={trigger[0]}>
              {trigger[1].displayName}
            </MenuItem>
          ))}
        </Input>

        <div>
          <Input
            label='Maximum concurrent runs'
            required={true}
            onChange={this.handleChange('maxConcurrentRuns')}
            value={maxConcurrentRuns}
            variant='outlined'
          />

          <div className={commonCss.flex}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={hasStartDate}
                  color='primary'
                  onClick={this.handleChange('hasStartDate')}
                />
              }
              label='Has start date'
            />
            <Input
              label='Start date'
              type='date'
              onChange={this.handleChange('startDate')}
              value={startDate}
              width={160}
              variant='outlined'
              InputLabelProps={{ classes: { outlined: css.noMargin }, shrink: true }}
              style={{ visibility: hasStartDate ? 'visible' : 'hidden' }}
            />
            <Separator />
            <Input
              label='Start time'
              type='time'
              onChange={this.handleChange('startTime')}
              value={startTime}
              width={120}
              variant='outlined'
              InputLabelProps={{ classes: { outlined: css.noMargin }, shrink: true }}
              style={{ visibility: hasStartDate ? 'visible' : 'hidden' }}
            />
          </div>

          <div className={commonCss.flex}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={hasEndDate}
                  color='primary'
                  onClick={this.handleChange('hasEndDate')}
                />
              }
              label='Has end date'
            />
            <Input
              label='End date'
              type='date'
              onChange={this.handleChange('endDate')}
              value={endDate}
              width={160}
              style={{ visibility: hasEndDate ? 'visible' : 'hidden' }}
              InputLabelProps={{ classes: { outlined: css.noMargin }, shrink: true }}
              variant='outlined'
            />
            <Separator />
            <Input
              label='End time'
              type='time'
              onChange={this.handleChange('endTime')}
              value={endTime}
              width={120}
              style={{ visibility: hasEndDate ? 'visible' : 'hidden' }}
              InputLabelProps={{ classes: { outlined: css.noMargin }, shrink: true }}
              variant='outlined'
            />
          </div>
          <span className={commonCss.flex}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={catchup}
                  color='primary'
                  onClick={this.handleChange('catchup')}
                />
              }
              label='Catchup'
            />
            <HelpButton
              helpText={
                <div>
                  <p>
                    Whether the recurring run should catch up if behind schedule. Defaults to true.
                  </p>
                  <p>
                    For example, if the recurring run is paused for a while and re-enabled
                    afterwards. If catchup=true, the scheduler will catch up on (backfill) each
                    missed interval. Otherwise, it only schedules the latest interval if more than
                    one interval is ready to be scheduled.
                  </p>
                  <p>
                    Usually, if your pipeline handles backfill internally, you should turn catchup
                    off to avoid duplicate backfill.
                  </p>
                </div>
              }
            />
          </span>

          <span className={commonCss.flex}>
            Run every
            {type === TriggerType.INTERVALED && (
              <div className={commonCss.flex}>
                <Separator />
                <Input
                  required={true}
                  type='number'
                  onChange={this.handleChange('intervalValue')}
                  value={intervalValue}
                  height={30}
                  width={65}
                  error={intervalValue < 1}
                  variant='outlined'
                />
              </div>
            )}
            <Separator />
            <Input
              required={true}
              select={true}
              onChange={this.handleChange('intervalCategory')}
              value={intervalCategory}
              height={30}
              width={95}
              variant='outlined'
            >
              {Object.keys(PeriodicInterval).map((interval, i) => (
                <MenuItem key={i} value={PeriodicInterval[interval]}>
                  {PeriodicInterval[interval] + (type === TriggerType.INTERVALED ? 's' : '')}
                </MenuItem>
              ))}
            </Input>
          </span>
        </div>

        {type === TriggerType.CRON && (
          <div>
            {intervalCategory === PeriodicInterval.WEEK && (
              <div>
                <span>On:</span>
                <FormControlLabel
                  control={
                    <Checkbox
                      checked={this._isAllDaysChecked()}
                      color='primary'
                      onClick={this._toggleCheckAllDays.bind(this)}
                    />
                  }
                  label='All'
                />
                <Separator />
                {['S', 'M', 'T', 'W', 'T', 'F', 'S'].map((day, i) => (
                  <Button
                    variant='fab'
                    mini={true}
                    key={i}
                    onClick={() => this._toggleDay(i)}
                    color={selectedDays[i] ? 'primary' : 'secondary'}
                  >
                    {day}
                  </Button>
                ))}
              </div>
            )}

            <div className={commonCss.flex}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={editCron}
                    color='primary'
                    onClick={this.handleChange('editCron')}
                  />
                }
                label={
                  <span>
                    Allow editing cron expression. ( format is specified{' '}
                    <a href='https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format'>
                      here
                    </a>
                    )
                  </span>
                }
              />
            </div>

            <Input
              label='cron expression'
              onChange={this.handleChange('cron')}
              value={cron}
              width={300}
              disabled={!editCron}
              variant='outlined'
            />

            <div>Note: Start and end dates/times are handled outside of cron.</div>
          </div>
        )}
      </div>
    );
  }

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

    const startDateTime = pickersToDate(hasStartDate, startDate, startTime);
    const endDateTime = pickersToDate(hasEndDate, endDate, endTime);

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
    return this.state.selectedDays.every(d => !!d);
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
