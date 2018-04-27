import 'app-datepicker/app-datepicker-dialog.html';
import 'neon-animation/web-animations.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-dropdown-menu/paper-dropdown-menu.html';
import 'paper-input/paper-input.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';
import 'paper-toggle-button/paper-toggle-button.html';
import 'polymer/polymer.html';

import * as Utils from '../../lib/utils';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import { PageElement } from '../../model/page_element';

import './pipeline-schedule.html';

const IMMEDIATELY = 'Run right away';
const RECURRING = 'Recurring';
const SPECIFIC_TIME = 'Run at a specific time';

const DATE_FORMAT_PATTERN = /^[1-2]\d{3}\/\d?\d\/\d?\d$/;

@customElement('pipeline-schedule')
export class PipelineSchedule extends Polymer.Element implements PageElement {

  @property({ notify: true, type: Boolean })
  public scheduleIsValid = true;

  @property({ type: Array })
  protected readonly _SCHEDULES = [
    IMMEDIATELY,
    SPECIFIC_TIME,
    RECURRING,
  ];

  @property({ type: Number })
  protected _scheduleTypeIndex = 0;

  @property({ type: Number })
  protected _runIntervalIndex = 0;

  @property({ type: Array })
  protected _runIntervals = [
    'hourly',
    'daily',
    'weekly',
    'monthly',
  ];

  @property({ type: String })
  protected _minStartDate = new Date().toLocaleDateString();

  @property({ type: String })
  protected _startDate = '';

  @property({ type: Boolean })
  protected _hasEndDate = false;

  @property({ type: String })
  protected _endDate = '';

  @property({ type: Array })
  protected _times: string[] = [];

  @property({ type: Number })
  protected _startTimeIndex = 0;

  @property({ type: Boolean })
  protected _isStartPM = true;

  @property({ type: Number })
  protected _endTimeIndex = 0;

  @property({ type: Boolean })
  protected _isEndPM = true;

  @property({ type: Boolean })
  protected _startTimeIsValid = true;

  @property({ type: String })
  protected _startTimeErrorMessage = '';

  @property({ type: Boolean })
  protected _endTimeIsValid = true;

  @property({ type: String })
  protected _endTimeErrorMessage = '';

  @property({ type: String })
  protected _crontab = '';

  public async load(_: string): Promise<void> {
    // TODO: disable or don't include invalid times.
    // Create an array with 12 hours + half hours for scheduling time.
    this._times.push('12:00');
    this._times.push('12:30');
    for (let i = 1; i < 12; i++) {
      this._times.push((i + '').padStart(2, '0') + ':00');
      this._times.push((i + '').padStart(2, '0') + ':30');
    }
  }

  public scheduleAsCrontab(): string {
    return this._crontab;
  }

  // TODO: Maybe use a polymer validator (property?) here?
  @observe('_scheduleTypeIndex, _startDate, _endDate, _hasEndDate,\
    _startTimeIndex, _endTimeIndex, _isStartPM, _isEndPM, _runIntervalIndex')
  protected _validateSchedule(): void {
    // Start and end time can't be invalid if we're running the job immediately.
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY) {
      this.scheduleIsValid = true;
    } else {
      // Update these properties to update validation UI.
      this._startTimeIsValid = this._isStartTimeValid();
      this._endTimeIsValid = this._isEndTimeValid();

      this.scheduleIsValid = this._startTimeIsValid && this._endTimeIsValid;
      if (this.scheduleIsValid) {
        this._updateCrontab();
      }
    }
  }

  // Ensure that end date is always >= start date.
  @observe('_startDate')
  protected _startDateChanged(newStartDate: string): void {
    if (newStartDate > this._endDate) {
      this._endDate = newStartDate;
      (this.$.endDatepicker as any).inputDate = this._endDate;
    }
  }

  // Launch datepicker dialog for starting date.
  protected _pickStartDate(): void {
    const datepicker = this.$.startDatepicker as any;
    datepicker.open();
  }

  // Launch datepicker dialog for ending date.
  protected _pickEndDate(): void {
    const datepicker = this.$.endDatepicker as any;
    datepicker.open();
  }

  // Show schedule inputs for a single run that will execute at a future time.
  protected _showSpecificTimeScheduleInputs(scheduleTypeIndex: number): boolean {
    return this._SCHEDULES[scheduleTypeIndex] === SPECIFIC_TIME ||
      this._showRecurringScheduleInputs(scheduleTypeIndex);
  }

  // Show schedule inputs for recurring runs.
  protected _showRecurringScheduleInputs(scheduleTypeIndex: number): boolean {
    return this._SCHEDULES[scheduleTypeIndex] === RECURRING;
  }

  // TODO: Do we need to localize time?
  private _updateCrontab(): void {
    const startDateTime = this._getStartDateTime();
    const minute = startDateTime.getMinutes();
    let hour = '*';
    let dayOfMonth = '*';
    const month = '*';
    const dayOfWeek = '*';
    switch (this._runIntervals[this._runIntervalIndex]) {
      case 'hourly':
        break;
      case 'daily':
        hour = '' + startDateTime.getHours();
        break;
      case 'weekly':
        // TODO: Add support for selecting days of the week.
        hour = '' + startDateTime.getHours();
        dayOfMonth += '/7';
        break;
      case 'monthly':
        hour = '' + startDateTime.getHours();
        dayOfMonth = '' + startDateTime.getDate();
        break;
      default:
        Utils.log.error('Invalid interval index:', this._runIntervalIndex);
    }
    this._crontab =
      minute + ' ' + hour + ' ' + dayOfMonth + ' ' + month + ' ' + dayOfWeek;
  }

  private _getStartDateTime(): Date {
    return this._toDate(this._startDate, this._startTimeIndex, this._isStartPM);
  }

  private _toDate(date: string, timeSelectorIndex: number, isPm: boolean): Date {
    const dateAsArray = date.split('/').map((s) => Number(s));
    // If/when start/end have different backing time arrays, this will need to
    // be updated.
    const timeAsArray =
      this._times[timeSelectorIndex].split(':').map((s) => Number(s));
    return new Date(
      dateAsArray[0],
      dateAsArray[1] - 1,
      dateAsArray[2],
      (timeAsArray[0] % 12) + (isPm ? 12 : 0),
      timeAsArray[1]);
  }

  private _isStartTimeValid(): boolean {
    // Check that start date is of form YYYY/MM/DD and is a valid date.
    if (!(this._startDate.match(DATE_FORMAT_PATTERN)) ||
        isNaN(Date.parse(this._startDate))) {
      this._startTimeErrorMessage = 'Must be valid date of the form YYYY/MM/DD';
      return false;
    }

    // Check that start time is not in the past.
    if (this._getStartDateTime() < new Date()) {
      this._startTimeErrorMessage = 'Start date/time cannot be in the past';
      return false;
    }

    return true;
  }

  private _isEndTimeValid(): boolean {
    // End time can't be invalid if we're running the job immediately or at a
    // specific time or if there's no end time.
    if (this._SCHEDULES[this._scheduleTypeIndex] === SPECIFIC_TIME ||
        !this._hasEndDate) {
      return true;
    }

    // Check that start date is of form YYYY/MM/DD and is a valid date.
    if (!(this._endDate.match(DATE_FORMAT_PATTERN)) ||
        isNaN(Date.parse(this._endDate))) {
      this._endTimeErrorMessage = 'Must be valid date of the form YYYY/MM/DD';
      return false;
    }

    const selectedEndDate =
      this._toDate(this._endDate, this._endTimeIndex, this._isEndPM);

    // If end time is in the past, it is invalid regardless of the start time.
    if (selectedEndDate < new Date()) {
      this._endTimeErrorMessage = 'End date/time cannot be in the past';
      return false;
    }

    // End date/time is invalid if it is earlier than start date/time.
    const selectedStartDate = this._getStartDateTime();
    if (selectedEndDate < selectedStartDate) {
      this._endTimeErrorMessage =
        'End date/time must be later than start date/time';
      return false;
    }
    return true;
  }
}
