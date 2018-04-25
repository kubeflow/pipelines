import 'app-datepicker/app-datepicker-dialog.html';
import 'neon-animation/web-animations.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-dropdown-menu/paper-dropdown-menu.html';
import 'paper-input/paper-input.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';
import 'paper-toggle-button/paper-toggle-button.html';
import 'polymer/polymer.html';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import { PageElement } from '../../model/page_element';

import './pipeline-schedule.html';

const IMMEDIATELY = 'Run right away';
const RECURRING = 'Recurring';
const SPECIFIC_TIME = 'Run at a specific time';

const RUN_INTERVALS_SINGULAR = [
  'hour',
  'day',
  'week',
  'month',
];

const DATE_FORMAT_PATTERN = /^[1-2]\d{3}\/\d?\d\/\d?\d$/;

const RUN_INTERVALS_PLURAL = RUN_INTERVALS_SINGULAR.map((i) => i + 's');

@customElement('pipeline-schedule')
export class PipelineSchedule extends Polymer.Element implements PageElement {

  @property({ notify: true, type: Boolean })
  public sheduleIsValid = true;

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

  @property({ type: Number })
  protected _runIntervalFrequency = 1;

  @property({ type: Array })
  protected _runIntervals = RUN_INTERVALS_SINGULAR;

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

  public async load(_: string) {
    // TODO: disable or don't include invalid times.
    // Create an array with 12 hours + half hours for scheduling time.
    this._times.push('12:00');
    this._times.push('12:30');
    for (let i = 1; i < 12; i++) {
      this._times.push((i + '').padStart(2, '0') + ':00');
      this._times.push((i + '').padStart(2, '0') + ':30');
    }
  }

  // TODO: Maybe use a polymer validator (property?) here?
  @observe('_scheduleTypeIndex, _startDate, _endDate, _hasEndDate,\
    _startTimeIndex, _endTimeIndex, _isStartPM, _isEndPM')
  protected _validateSchedule() {
    // Update these properties to update validation UI.
    this._startTimeIsValid = this._isStartTimeValid();
    this._endTimeIsValid = this._isEndTimeValid();

    this.sheduleIsValid = this._startTimeIsValid && this._endTimeIsValid;
  }

  // Ensure that end date is always >= start date.
  @observe('_startDate')
  protected _startDateChanged(newStartDate: string) {
    if (newStartDate > this._endDate) {
      this._endDate = newStartDate;
      (this.$.endDatepicker as any).inputDate = this._endDate;
    }
  }

  // This takes a string because that's what the observer passes it.
  // We immediately convert to a number so that it's clear that it represents a
  // number.
  @observe('_runIntervalFrequency')
  protected _runIntervalFrequencyChanged(newFrequencyString: string) {
    const newFrequency = Number(newFrequencyString);
    const root = this.shadowRoot as ShadowRoot;
    if (root.querySelector('#runIntervalDropdown')) {
      this._runIntervals =
        newFrequency === 1 ? RUN_INTERVALS_SINGULAR : RUN_INTERVALS_PLURAL;
      (root.querySelector('#runIntervalDropdown') as any).value =
        this._runIntervals[this._runIntervalIndex];
    }
  }

  // Launch datepicker dialog for starting date.
  protected _pickStartDate() {
    const datepicker = this.$.startDatepicker as any;
    datepicker.open();
  }

  // Launch datepicker dialog for ending date.
  protected _pickEndDate() {
    const datepicker = this.$.endDatepicker as any;
    datepicker.open();
  }

  // Show schedule inputs for a single run that will execute at a future time.
  protected _showSpecificTimeScheduleInputs(scheduleTypeIndex: number) {
    return this._SCHEDULES[scheduleTypeIndex] === SPECIFIC_TIME ||
      this._showRecurringScheduleInputs(scheduleTypeIndex);
  }

  // Show schedule inputs for recurring runs.
  protected _showRecurringScheduleInputs(scheduleTypeIndex: number) {
    return this._SCHEDULES[scheduleTypeIndex] === RECURRING;
  }

  private _toDate(date: string, timeSelectorIndex: number,
                  isPm: boolean): Date {
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
    // Start time can't be invalid if we're running the job immediately.
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY) {
      return true;
    }

    // Check that start date is of form YYYY/MM/DD and is a valid date.
    if (!(this._startDate.match(DATE_FORMAT_PATTERN)) ||
        isNaN(Date.parse(this._startDate))) {
      this._startTimeErrorMessage = 'Must be valid date of the form YYYY/MM/DD';
      return false;
    }

    // Check that start time is not in the past.
    if (this._toDate(this._startDate, this._startTimeIndex, this._isStartPM) <
        new Date()) {
      this._startTimeErrorMessage = 'Start date/time cannot be in the past';
      return false;
    }

    return true;
  }

  private _isEndTimeValid(): boolean {
    // End time can't be invalid if we're running the job immediately or at a
    // specific time or if there's no end time.
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY ||
        this._SCHEDULES[this._scheduleTypeIndex] === SPECIFIC_TIME ||
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
    const selectedStartDate =
      this._toDate(this._startDate, this._startTimeIndex, this._isStartPM);
    if (selectedEndDate < selectedStartDate) {
      this._endTimeErrorMessage =
        'End date/time must be later than start date/time';
      return false;
    }
    return true;
  }
}
