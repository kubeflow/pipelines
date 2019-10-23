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

import * as React from 'react';
import Button from '@material-ui/core/Button';
import Checkbox from '@material-ui/core/Checkbox';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Input from '../atoms/Input';
import MenuItem from '@material-ui/core/MenuItem';
import Separator from '../atoms/Separator';
import { commonCss } from '../Css';
import { dateToPickerFormat } from '../lib/TriggerUtils';
import {
  PeriodicInterval,
  TriggerType,
  triggers,
  buildCron,
  pickersToDate,
  buildTrigger,
} from '../lib/TriggerUtils';
import { ApiTrigger } from '../apis/job';
import { stylesheet } from 'typestyle';

interface TriggerProps {
  onChange?: (trigger?: ApiTrigger, maxConcurrentRuns?: string) => void;
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
}

const css = stylesheet({
  noMargin: {
    margin: 0,
  },
});

export default class Trigger extends React.Component<TriggerProps, TriggerState> {
  constructor(props: any) {
    super(props);

    const now = new Date();
    const inAWeek = new Date(
      now.getFullYear(),
      now.getMonth(),
      now.getDate() + 7,
      now.getHours(),
      now.getMinutes(),
    );
    const [startDate, startTime] = dateToPickerFormat(now);
    const [endDate, endTime] = dateToPickerFormat(inAWeek);

    this.state = {
      cron: '',
      editCron: false,
      endDate,
      endTime,
      hasEndDate: false,
      hasStartDate: false,
      intervalCategory: PeriodicInterval.MINUTE,
      intervalValue: 1,
      maxConcurrentRuns: '10',
      selectedDays: new Array(7).fill(true),
      startDate,
      startTime,
      type: TriggerType.INTERVALED,
    };
  }

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
              {Object.keys(PeriodicInterval).map((interval: PeriodicInterval, i) => (
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
                    <a href='https://godoc.org/github.com/robfig/cron#hdr-CRON_Expression_Format'>
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
      this._updateTrigger.bind(this),
    );
  };

  private _updateTrigger(): void {
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
          this.props.onChange(trigger, trigger ? this.state.maxConcurrentRuns : undefined);
        }
      },
    );
  }

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
      this._updateTrigger.bind(this),
    );
  }
}
