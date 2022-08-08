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

import Button from '@material-ui/core/Button';
import Checkbox from '@material-ui/core/Checkbox';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import MenuItem from '@material-ui/core/MenuItem';
import * as React from 'react';
import { stylesheet } from 'typestyle';
import { ApiTrigger } from '../apis/job';
import { HelpButton } from '../atoms/HelpButton';
import { ExternalLink } from '../atoms/ExternalLink';
import Input from '../atoms/Input';
import Separator from '../atoms/Separator';
import { color, commonCss } from '../Css';
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
import { t } from 'i18next';

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
  startTimeMessage: string;
  endTimeMessage: string;
}

const css = stylesheet({
  noMargin: {
    margin: 0,
  },
  alert: {
    color: color.alert,
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
      startTimeMessage,
      endTimeMessage,
    } = this.state;

    return (
      <div>
        <Input
          select={true}
          label={t('trigger.type')}
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
            label={t('trigger.max_concurrent_count')}
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
              label={t('trigger.has_start_date')}
            />
            <Input
              label={t('trigger.start_date')}
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
              label={t('trigger.start_time')}
              type='time'
              onChange={this.handleChange('startTime')}
              value={startTime}
              width={120}
              variant='outlined'
              InputLabelProps={{ classes: { outlined: css.noMargin }, shrink: true }}
              style={{ visibility: hasStartDate ? 'visible' : 'hidden' }}
            />
          </div>
          <div
            data-testid='startTimeMessage'
            className={css.alert}
            style={{ visibility: hasStartDate ? 'visible' : 'hidden' }}
          >
            {startTimeMessage}
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
              label={t('trigger.has_end_date')}
            />
            <Input
              label={t('trigger.end_date')}
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
              label={t('trigger.end_time')}
              type='time'
              onChange={this.handleChange('endTime')}
              value={endTime}
              width={120}
              style={{ visibility: hasEndDate ? 'visible' : 'hidden' }}
              InputLabelProps={{ classes: { outlined: css.noMargin }, shrink: true }}
              variant='outlined'
            />
          </div>
          <div
            data-testid='endTimeMessage'
            className={css.alert}
            style={{ visibility: hasEndDate ? 'visible' : 'hidden' }}
          >
            {endTimeMessage}
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
              label={t('trigger.catchup')}
            />
            <HelpButton
              helpText={
                <div>
                  <p>
                    {t('trigger.help_text_part1')}
                  </p>
                  <p>
                    {t('trigger.help_text_part2')}
                  </p>
                  <p>
                    {t('trigger.help_text_part3')}
                  </p>
                </div>
              }
            />
          </span>

          <span className={commonCss.flex}>
            {t('trigger.run_every')}
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
                    Allow editing cron expression. (format is specified{' '}
                    <ExternalLink href='https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format'>
                      here
                    </ExternalLink>
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

    var startDateTime: Date | undefined = undefined;
    var endDateTime: Date | undefined = undefined;
    var startTimeMessage = '';
    var endTimeMessage = '';

    try {
      if (hasStartDate) {
        startDateTime = pickersToDate(hasStartDate, startDate, startTime);
      }
    } catch (e) {
      if (e.message === 'Invalid picker format') {
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
      if (e.message === 'Invalid picker format') {
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
