import 'app-datepicker/app-datepicker-dialog.html';
import 'neon-animation/web-animations.html';
import 'paper-button/paper-button.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-dropdown-menu/paper-dropdown-menu.html';
import 'paper-input/paper-input.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';
import 'paper-toggle-button/paper-toggle-button.html';
import 'polymer/polymer.html';

import * as Utils from '../../lib/utils';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';

import './pipeline-schedule.html';

const IMMEDIATELY = 'Run right away';
const RECURRING = 'Recurring';
const SPECIFIC_TIME = 'Run at a specific time';

const DATE_FORMAT_PATTERN = /^[1-2]\d{3}\/\d?\d\/\d?\d$/;

enum Intervals {
  MINUTE = 'every minute',
  HOURLY = 'hourly',
  DAILY = 'daily',
  WEEKLY = 'weekly',
  MONTHLY = 'monthly',
}

@customElement('pipeline-schedule')
export class PipelineSchedule extends Polymer.Element {

  @property({ notify: true, type: Boolean })
  public scheduleIsValid = true;

  @property({ type: Array })
  protected readonly _SCHEDULES = [
    IMMEDIATELY,
    RECURRING,
  ];

  @property({ type: Number })
  protected _scheduleTypeIndex = 0;

  @property({ type: Number })
  protected _runIntervalIndex = 1;

  @property({ type: Array })
  protected _runIntervals = [
    Intervals.MINUTE,
    Intervals.HOURLY,
    Intervals.DAILY,
    Intervals.WEEKLY,
    Intervals.MONTHLY,
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

  @property({ computed: '_checkIfAllActive(_weekdays.*)', type: Boolean })
  protected _allDaysOfWeekActive = true;

  @property({ type: Array })
  protected _weekdays = [
    { day: 'S', active: true },
    { day: 'M', active: true },
    { day: 'T', active: true },
    { day: 'W', active: true },
    { day: 'T', active: true },
    { day: 'F', active: true },
    { day: 'S', active: true }
  ];

  @property({ type: Boolean })
  protected _enableWeekdayButtons = true;

  @property({ type: Boolean })
  protected _weekdaySelectionIsValid = true;

  constructor() {
    super();
    // TODO: disable or don't include invalid times.
    // Create an array with 12 hours + half hours for scheduling time.
    this._times.push('12:00');
    this._times.push('12:30');
    for (let i = 1; i < 12; i++) {
      this._times.push((i + '').padStart(2, '0') + ':00');
      this._times.push((i + '').padStart(2, '0') + ':30');
    }
  }

  // Backend expects the crontab in UTC.
  public scheduleAsUTCCrontab(): string {
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY) {
      return '';
    }
    // TODO: verify specific time is working as intended.
    const startDateTime = this._getStartDateTime();
    return this._generateCrontab(
        startDateTime.getUTCDate(),
        startDateTime.getUTCHours(),
        startDateTime.getUTCMinutes());
  }

  // TODO: Maybe use a polymer validator (property?) here?
  @observe('_scheduleTypeIndex, _startDate, _endDate, _hasEndDate,\
    _startTimeIndex, _endTimeIndex, _isStartPM, _isEndPM, _runIntervalIndex,\
    _weekdays.*')
  protected _validateSchedule(): void {
    // Start and end time can't be invalid if we're running the job immediately.
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY) {
      this.scheduleIsValid = true;
    } else {
      // Update these properties to update validation UI.
      this._startTimeIsValid = this._isStartTimeValid();
      this._endTimeIsValid = this._isEndTimeValid();

      // Weekday selection is valid if interval is not weekly or any weekday is
      // selected.
      this._weekdaySelectionIsValid =
          this._runIntervals[this._runIntervalIndex] !== Intervals.WEEKLY ||
          this._weekdays.map((w) => w.active).reduce((prev, cur) => prev || cur);

      this.scheduleIsValid = this._startTimeIsValid && this._endTimeIsValid &&
          this._weekdaySelectionIsValid;

      if (this.scheduleIsValid) {
        this._updateDisplayCrontab();
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

  // Update all-weekdays checkbox when a weekday button is pressed.
  protected _checkIfAllActive(): boolean {
    return this._weekdays.map((w) => w.active).reduce((prev, cur) => prev && cur);
  }

  // Update all weekday buttons when all-weekdays checkbox is (un)checked.
  protected _selectAllWeekdaysCheckboxChanged(): void {
    const root = this.shadowRoot as ShadowRoot;
    // If this function is called on-checked-changed, the property in this
    // class won't have actually been updated yet, so we have to get it this
    // way.
    const allWeekdaysCheckbox = root.querySelector('#allWeekdaysCheckbox');
    if (allWeekdaysCheckbox) {
      const allWeekdaysCheckboxChecked = (allWeekdaysCheckbox as PaperCheckboxElement).checked;
      this._weekdays.forEach((_, i) =>
          this.set('_weekdays.' + i + '.active', allWeekdaysCheckboxChecked));
    }
  }

  @observe('_runIntervalIndex')
  protected _updateWeekdayButtonEnabledState(): void {
    // Weekdays are only enabled if interval is 'weekly'.
    this._enableWeekdayButtons =
        this._runIntervals[this._runIntervalIndex] === Intervals.WEEKLY;
    if (!this._enableWeekdayButtons) {
      // Check 'All weekdays' checkbox and update individual weekday buttons.
      this._allDaysOfWeekActive = true;
      this._selectAllWeekdaysCheckboxChanged();
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

  // We don't simply take a Date object here because the crontab we send to the
  // backend needs to be UTC, and Date objects are automatically local.
  private _generateCrontab(targetDateDay: number, targetDateHours: number,
      targetDateMinutes: number): string {
    // The default values here correspond to 'run at second 0 of every minute'
    const second = '0';
    let minute = '*';
    let hour = '*';
    let dayOfMonth = '*';
    const month = '*';
    let dayOfWeek = '?';
    switch (this._runIntervals[this._runIntervalIndex]) {
      case Intervals.MINUTE:
        break;
      case Intervals.HOURLY:
        minute = '' + targetDateMinutes;
        break;
      case Intervals.DAILY:
        minute = '' + targetDateMinutes;
        hour = '' + targetDateHours;
        break;
      case Intervals.WEEKLY:
        minute = '' + targetDateMinutes;
        hour = '' + targetDateHours;
        dayOfMonth = '?';
        if (this._checkIfAllActive()) {
          dayOfWeek = '*';
        } else {
          // Convert weekdays to array of indices of active days and join them.
          dayOfWeek = this._weekdays.reduce(
              (result: number[], day, i) => {
                if (day.active) { result.push(i); }
                return result;
              },
              []).join(',');
        }
        break;
      case Intervals.MONTHLY:
        minute = '' + targetDateMinutes;
        hour = '' + targetDateHours;
        dayOfMonth = '' + targetDateDay;
        break;
      default:
        Utils.log.error('Invalid interval index:', this._runIntervalIndex);
    }
    return [ second, minute, hour, dayOfMonth, month, dayOfWeek ].join(' ');
  }

  @observe('_weekdays.*')
  private _updateDisplayCrontab(): void {
    const startDateTime = this._getStartDateTime();
    this._crontab = this._generateCrontab(
        startDateTime.getDate(),
        startDateTime.getHours(),
        startDateTime.getMinutes());
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
