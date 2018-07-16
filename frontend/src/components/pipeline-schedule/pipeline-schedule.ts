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
import { CronSchedule, Trigger } from '../../api/pipeline';
import { DateTimePicker } from '../date-time-picker/date-time-picker';

import './pipeline-schedule.html';

const IMMEDIATELY = 'Run right away';
const CRON = 'Cron';

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

  // Visible for testing
  @property({ type: Array })
  public _runIntervals = [
    Intervals.MINUTE,
    Intervals.HOURLY,
    Intervals.DAILY,
    Intervals.WEEKLY,
    Intervals.MONTHLY,
  ];

  @property({ type: Array })
  protected readonly _SCHEDULES = [
    IMMEDIATELY,
    CRON,
  ];

  @property({ type: Number })
  protected _scheduleTypeIndex = 0;

  // Set default interval to 'hourly'
  @property({ type: Number })
  protected _runIntervalIndex = 1;

  @property({ type: String })
  protected _crontab = '';

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

  // TODO: write getters for the date-time pickers.

  public get scheduleTypeDropdown(): PaperDropdownMenuElement {
    return this.$.scheduleTypeDropdown as PaperDropdownMenuElement;
  }

  public get scheduleTypeListbox(): PaperListboxElement {
    return this.$.scheduleTypeListbox as PaperListboxElement;
  }

  public get startDateTimePicker(): DateTimePicker|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#startDateTimePicker') : null;
  }

  public get endDateTimePicker(): DateTimePicker|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#endDateTimePicker') : null;
  }

  public get intervalDropdown(): PaperDropdownMenuElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#intervalDropdown') : null;
  }

  public get intervalListbox(): PaperListboxElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#intervalListbox') : null;
  }

  public get allWeekdaysCheckbox(): PaperCheckboxElement|null {
    return this.shadowRoot ? this.shadowRoot.querySelector('#allWeekdaysCheckbox') : null;
  }

  public toTrigger(): Trigger|null {
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY) {
      return null;
    }

    const cronSchedule = new CronSchedule(this._crontab);
    const startTime = this._startTime();
    const endTime = this._endTime();
    if (startTime) {
      cronSchedule.start_time = startTime;
    }
    if (endTime) {
      cronSchedule.end_time = endTime;
    }
    const trigger = new Trigger();
    trigger.cron_schedule = cronSchedule;
    return trigger;
  }

  // TODO: Maybe use a polymer validator (property?) here?
  // Our requirements necessitate that we know whether each date-time is valid, invalid, or hidden
  // as distinct states.
  @observe('_scheduleTypeIndex, _runIntervalIndex, _startIsValid, _startDate, _endDate,\
      _endIsValid, _weekdays.*')
  protected _validateSchedule(): void {
    this._errorMsg = '';
    // Start and end time can't be invalid if we're running the job immediately.
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY) {
      this.scheduleIsValid = true;
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

      // Weekday selection is valid if interval is not weekly or any weekday is
      // selected.
      this._weekdaySelectionIsValid =
          this._runIntervals[this._runIntervalIndex] !== Intervals.WEEKLY ||
          this._weekdays.map((w) => w.active).reduce((prev, cur) => prev || cur);

      this.scheduleIsValid = startAndEndAreValid && this._weekdaySelectionIsValid;

      if (this.scheduleIsValid) {
        this._updateDisplayCrontab();
      }
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
    if (root) {
      const allWeekdaysCheckbox = root.querySelector('#allWeekdaysCheckbox');
      if (allWeekdaysCheckbox) {
        const allWeekdaysCheckboxChecked = (allWeekdaysCheckbox as PaperCheckboxElement).checked;
        this._weekdays.forEach(
            (_, i) => this.set('_weekdays.' + i + '.active', allWeekdaysCheckboxChecked));
      }
    }
  }

  @observe('_runIntervalIndex')
  protected _updateWeekdayButtonEnabledState(): void {
    // Weekdays are only enabled if interval is 'weekly'.
    this._enableWeekdayButtons = this._runIntervals[this._runIntervalIndex] === Intervals.WEEKLY;
    if (!this._enableWeekdayButtons) {
      // Check 'All weekdays' checkbox and update individual weekday buttons.
      this._allDaysOfWeekActive = true;
      this._selectAllWeekdaysCheckboxChanged();
    }
  }

  // Show Cron schedule UI
  protected _showCronScheduleInputs(scheduleTypeIndex: number): boolean {
    return this._SCHEDULES[scheduleTypeIndex] === CRON;
  }

  private _startTime(): string {
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY || !this._startDate) {
      return '';
    }
    return (this.$.startDatepicker as DateTimePicker).dateTimeAsIsoString();
  }

  private _endTime(): string {
    if (this._SCHEDULES[this._scheduleTypeIndex] === IMMEDIATELY || !this._endDate) {
      return '';
    }
    return (this.$.endDatepicker as DateTimePicker).dateTimeAsIsoString();
  }

  // The crontab we send to the backend needs to be UTC.
  private _generateCrontab(): string {
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
    switch (this._runIntervals[this._runIntervalIndex]) {
      case Intervals.MINUTE:
        break;
      case Intervals.HOURLY:
        minute = targetMinutes || minute;
        break;
      case Intervals.DAILY:
        minute = targetMinutes || minute;
        hour = targetHours || hour;
        break;
      case Intervals.WEEKLY:
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
      case Intervals.MONTHLY:
        minute = targetMinutes || minute;
        hour = targetHours || hour;
        dayOfMonth = targetDayOfMonth || dayOfMonth;
        break;
      default:
        Utils.log.error('Invalid interval index:', this._runIntervalIndex);
    }
    return [ second, minute, hour, dayOfMonth, month, dayOfWeek ].join(' ');
  }

  @observe('_weekdays.*')
  private _updateDisplayCrontab(): void {
    this._crontab = this._generateCrontab();
  }
}
