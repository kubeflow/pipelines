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
import { CronSchedule, PeriodicSchedule, Trigger } from '../../api/job';
import { DateTimePicker } from '../date-time-picker/date-time-picker';

import './job-schedule.html';

const IMMEDIATELY = 'Run right away';
const PERIODIC = 'Periodic';
const CRON = 'Cron';

enum CronIntervals {
  MINUTE = 'every minute',
  HOURLY = 'hourly',
  DAILY = 'daily',
  WEEKLY = 'weekly',
  MONTHLY = 'monthly',
}

enum PeriodicIntervals {
  MINUTES = 'minutes',
  HOURS = 'hours',
  DAYS = 'days',
  WEEKS = 'weeks',
  MONTHS = 'months',
}

@customElement('job-schedule')
export class JobSchedule extends Polymer.Element {

  @property({ notify: true, type: Boolean })
  public scheduleIsValid = true;

  // Visible for testing
  @property({ type: Array })
  public _cronIntervals = [
    CronIntervals.MINUTE,
    CronIntervals.HOURLY,
    CronIntervals.DAILY,
    CronIntervals.WEEKLY,
    CronIntervals.MONTHLY,
  ];

  // Visible for testing
  @property({ type: Array })
  public _periodicIntervals = [
    PeriodicIntervals.MINUTES,
    PeriodicIntervals.HOURS,
    PeriodicIntervals.DAYS,
    PeriodicIntervals.WEEKS,
    PeriodicIntervals.MONTHS,
  ];

  @property({ type: Array })
  protected readonly _SCHEDULES = [
    IMMEDIATELY,
    PERIODIC,
    CRON,
  ];

  @property({ type: Number })
  protected _scheduleTypeIndex = 0;

  @property({ type: Boolean })
  protected _maxConcurrentRunsIsValid = true;

  // Set default interval to 'hourly'
  @property({ type: Number })
  protected _intervalIndex = 1;

  @property({ type: Number })
  protected _frequency = 1;

  @property({ type: Number })
  protected _maxConcurrentRuns = 10;

  @property({ type: String })
  protected _cronExpression = '';

  @property({ type: Boolean })
  protected _allowEditingCronExpression = false;

  @property({ type: String })
  protected _errorMsg = '';

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

  @property({ type: Boolean })
  protected _startIsValid = true;

  @property({ type: Boolean })
  protected _endIsValid = true;

  @property({ type: Object })
  protected _startDate: Date|null = null;

  @property({ type: Object })
  protected _endDate: Date|null = null;

  public get startDateTimePicker(): DateTimePicker|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#startDateTimePicker') : null;
  }

  public get endDateTimePicker(): DateTimePicker|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#endDateTimePicker') : null;
  }

  public get scheduleTypeDropdown(): PaperDropdownMenuElement {
    return this.$.scheduleTypeDropdown as PaperDropdownMenuElement;
  }

  public get scheduleTypeListbox(): PaperListboxElement {
    return this.$.scheduleTypeListbox as PaperListboxElement;
  }

  public get maxConcurrentRunsInput(): PaperInputElement {
    return this.$.maxConcurrentRuns as PaperInputElement;
  }

  public get cronIntervalDropdown(): PaperDropdownMenuElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#cronIntervalDropdown') : null;
  }

  public get cronIntervalListbox(): PaperListboxElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#cronIntervalListbox') : null;
  }

  public get allowEditingCronCheckbox(): PaperCheckboxElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#allowEditingCron') : null;
  }

  public get cronExpressionInput(): PaperInputElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#cronExpression') : null;
  }

  public get periodFrequencyInput(): PaperInputElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#frequency') : null;
  }

  public get periodicIntervalDropdown(): PaperDropdownMenuElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#periodicIntervalDropdown') : null;
  }

  public get periodicIntervalListbox(): PaperListboxElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#periodicIntervalListbox') : null;
  }

  public get allWeekdaysCheckbox(): PaperCheckboxElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#allWeekdaysCheckbox') : null;
  }

  public get maxConcurrentRuns(): number {
    return this._maxConcurrentRuns;
  }

  public toTrigger(): Trigger|null {
    if (this._SCHEDULES[this._scheduleTypeIndex] === PERIODIC) {
      return new Trigger(
          new PeriodicSchedule(this._getPeriodInSeconds(), this._startTime(), this._endTime()));
    }

    if (this._SCHEDULES[this._scheduleTypeIndex] === CRON) {
      return new Trigger(
          new CronSchedule(this._cronExpression, this._startTime(), this._endTime()));
    }

    return null;
  }

  // TODO: Maybe use a polymer validator (property?) here?
  // Our requirements necessitate that we know whether each date-time is valid, invalid, or hidden
  // as distinct states.
  @observe('_scheduleTypeIndex, _intervalIndex, _startIsValid, _startDate, _endDate,\
      _endIsValid, _maxConcurrentRunsIsValid, _weekdays.*')
  protected _validateSchedule(): void {
    this._errorMsg = '';
    // Start and end time can't be invalid if we're starting the runs immediately.
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY) {
      this.scheduleIsValid = this._maxConcurrentRunsIsValid;
    } else {
      let startAndEndAreValid = this._startIsValid && this._endIsValid;
      if (startAndEndAreValid) {
        // If start and end are considered valid, then this only needs to be false if both start and
        // end are also defined, and end is less than start.
        if (this._startDate && this._endDate) {
          startAndEndAreValid = this._startDate <= this._endDate;
          if (!startAndEndAreValid) {
            this._errorMsg = 'End date must be later than start date!';
          }
        }
      }

      this.scheduleIsValid = startAndEndAreValid && this._maxConcurrentRunsIsValid;

      if (this._SCHEDULES[this._scheduleTypeIndex] === CRON) {
        // Weekday selection is valid if interval is not weekly or any weekday is selected.
        this._weekdaySelectionIsValid =
            this._cronIntervals[this._intervalIndex] !== CronIntervals.WEEKLY ||
            this._weekdays.map((w) => w.active).reduce((prev, cur) => prev || cur);

        this.scheduleIsValid = this.scheduleIsValid && this._weekdaySelectionIsValid;

        if (this.scheduleIsValid && !this._allowEditingCronExpression) {
          this._updateDisplayCronExpression();
        }
      }
    }
  }

  @observe('_maxConcurrentRuns')
  protected _validateMaxConcurrentRuns(): void {
    this._maxConcurrentRunsIsValid = this._maxConcurrentRuns > 0 && this._maxConcurrentRuns <= 100;
  }

  // Update all-weekdays checkbox when a weekday button is pressed.
  protected _checkIfAllActive(): boolean {
    return this._weekdays.map((w) => w.active).reduce((prev, cur) => prev && cur);
  }

  // Update all weekday buttons when all-weekdays checkbox is (un)checked.
  protected _selectAllWeekdaysCheckboxChanged(): void {
    const root = this.shadowRoot as ShadowRoot;
    // If this function is called on-checked-changed, the property in this class won't have actually
    // been updated yet, so we have to get it this way.
    if (root) {
      const allWeekdaysCheckbox = root.querySelector('#allWeekdaysCheckbox');
      if (allWeekdaysCheckbox) {
        const allWeekdaysCheckboxChecked = (allWeekdaysCheckbox as PaperCheckboxElement).checked;
        this._weekdays.forEach(
            (_, i) => this.set('_weekdays.' + i + '.active', allWeekdaysCheckboxChecked));
      }
    }
  }

  protected _disableWeekdayButtons(
      enableWeekdayButtons: boolean,
      allowEditingCronExpression: boolean): boolean {
    // If a user opts to manually enter the cron expression, then we disable the other UI we provide
    // for constructing simple cron expressions.
    return !enableWeekdayButtons || allowEditingCronExpression;
  }

  @observe('_intervalIndex')
  protected _updateWeekdayButtonEnabledState(): void {
    // Weekdays are only enabled if interval is 'weekly'.
    this._enableWeekdayButtons =
        this._cronIntervals[this._intervalIndex] === CronIntervals.WEEKLY;
    if (!this._enableWeekdayButtons) {
      // Check 'All weekdays' checkbox and update individual weekday buttons.
      this._allDaysOfWeekActive = true;
      this._selectAllWeekdaysCheckboxChanged();
    }
  }

  // Show date/time inputs which are shared by Periodic and Cron schedules.
  protected _showDateTimePickers(scheduleTypeIndex: number): boolean {
    return this._showPeriodicInputs(scheduleTypeIndex) || this._showCronInputs(scheduleTypeIndex);
  }

  // Show schedule inputs for recurring runs specified by intervals.
  protected _showPeriodicInputs(scheduleTypeIndex: number): boolean {
    return this._SCHEDULES[scheduleTypeIndex] === PERIODIC;
  }

  // Show schedule inputs for recurring runs specified by cron.
  protected _showCronInputs(scheduleTypeIndex: number): boolean {
    return this._SCHEDULES[scheduleTypeIndex] === CRON;
  }

  // Disable tabbing to the cron expression input field and change its style if it is not editable.
  // We use CSS rather than the paper-input's "disabled" property because of an issue with how
  // Polymer treats tabindex when the disabled property changes.
  // See: https://github.com/PolymerElements/iron-behaviors/pull/83
  @observe('_allowEditingCronExpression')
  protected _updateCronInputDisabledState(): void {
    if (this.cronExpressionInput) {
      if (this._allowEditingCronExpression) {
        this.cronExpressionInput.classList.remove('disabled');
        this.cronExpressionInput.setAttribute('tabindex', '0');
      } else {
        this.cronExpressionInput.classList.add('disabled');
        this.cronExpressionInput.setAttribute('tabindex', '-1');
        this._updateDisplayCronExpression();
      }
    }
  }

  private _startTime(): string {
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY ||
        !this._startDate || !this.startDateTimePicker) {
      return '';
    }
    return this.startDateTimePicker.dateTimeAsIsoString();
  }

  private _endTime(): string {
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY ||
        !this._endDate || !this.endDateTimePicker) {
      return '';
    }
    return this.endDateTimePicker.dateTimeAsIsoString();
  }

  private _getPeriodInSeconds(): number {
    let intervalSeconds = 0;
    switch (this._periodicIntervals[this._intervalIndex]) {
      case PeriodicIntervals.MINUTES:
        intervalSeconds = 60;
        break;
      case PeriodicIntervals.HOURS:
        intervalSeconds = 60 * 60;
        break;
      case PeriodicIntervals.DAYS:
        intervalSeconds = 60 * 60 * 24;
        break;
      case PeriodicIntervals.WEEKS:
        intervalSeconds = 60 * 60 * 24 * 7;
        break;
      case PeriodicIntervals.MONTHS:
        intervalSeconds = 60 * 60 * 24 * 30;
        break;
      default:
        Utils.log.error('Invalid interval index:', this._intervalIndex);
        return -1;
    }
    return intervalSeconds * this._frequency;
  }

  // The cron expression we send to the backend needs to be UTC.
  private _generateCronExpression(): string {
    let targetDayOfMonth = '0';
    let targetHours = '0';
    let targetMinutes = '0';
    if (this._startDate && this._startIsValid) {
      targetDayOfMonth = '' + this._startDate.getDate();
      targetHours = '' + this._startDate.getHours();
      targetMinutes = '' + this._startDate.getMinutes();
    }

    // The default values here correspond to 'run at second 0 of every minute'
    const second = '0';
    let minute = '*';
    let hour = '*';
    let dayOfMonth = '*';
    const month = '*';
    let dayOfWeek = '?';
    switch (this._cronIntervals[this._intervalIndex]) {
      case CronIntervals.MINUTE:
        break;
      case CronIntervals.HOURLY:
        minute = targetMinutes || minute;
        break;
      case CronIntervals.DAILY:
        minute = targetMinutes || minute;
        hour = targetHours || hour;
        break;
      case CronIntervals.WEEKLY:
        minute = targetMinutes || minute;
        hour = targetHours || hour;
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
      case CronIntervals.MONTHLY:
        minute = targetMinutes || minute;
        hour = targetHours || hour;
        dayOfMonth = targetDayOfMonth || dayOfMonth;
        break;
      default:
        Utils.log.error('Invalid interval index:', this._intervalIndex);
    }
    return [ second, minute, hour, dayOfMonth, month, dayOfWeek ].join(' ');
  }

  @observe('_weekdays.*')
  private _updateDisplayCronExpression(): void {
    this._cronExpression = this._generateCronExpression();
  }
}
