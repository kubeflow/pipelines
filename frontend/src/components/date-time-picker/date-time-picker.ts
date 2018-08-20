import 'app-datepicker/app-datepicker-dialog.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-input/paper-input.html';
import 'paper-toggle-button/paper-toggle-button.html';

import { customElement, property } from 'polymer-decorators/src/decorators';

import './date-time-picker.html';

const DATE_FORMAT_PATTERN = /^[1-2]\d{3}\/\d?\d\/\d?\d$/;

@customElement('date-time-picker')
export class DateTimePicker extends Polymer.Element {

  @property({ type: String })
  public name = '';

  @property({
    computed: '_validate(_date, hasDate, _hours, _minutes, _isPm)',
    notify: true,
    readOnly: true,
    type: Boolean,
  })
  public valid = true;

  @property({ notify: true, type: Object })
  public datetime: Date | undefined = undefined;

  @property({ type: String })
  protected _minDate = new Date().toLocaleDateString();

  @property({ notify: true, type: Boolean })
  protected hasDate = false;

  @property({ type: String })
  protected _date = '';

  @property({ type: Boolean })
  protected _isPm = true;

  @property({ type: String })
  protected _hours: string;

  @property({ type: String })
  protected _minutes: string;

  @property({ type: String })
  protected _errorMessage = '';

  constructor() {
    super();
    // Set starting time to 10 minutes in future to avoid issues with starting in the past.
    const soon = new Date(Date.now() + 1000 * 60 * 10);
    this._isPm = soon.getHours() > 11;
    this._hours = ((soon.getHours() + 11) % 12) + 1 + '';
    this._minutes = (soon.getMinutes() < 10 ? '0' : '') + soon.getMinutes();
  }

  public get useDateTimeCheckbox(): PaperCheckboxElement {
    return this.$.useDateTimeCheckbox as PaperCheckboxElement;
  }

  public dateTimeAsIsoString(): string {
    return this.datetime ? this.datetime.toISOString() : '';
  }

  // Launch datepicker dialog for starting date.
  protected _launchDatePicker(): void {
    const datepicker = this.$.datepicker as any;
    datepicker.open();
  }

  protected _highlightField(ev: FocusEvent): void {
    ((ev.srcElement as PaperInputElement).$.input as IronInputElement).inputElement.select();
  }

  protected _jumpToFirstTimeField(): void {
    if (this.hasDate) {
      (this.shadowRoot!.querySelector('#hours') as PaperInputElement).focus();
    }
  }

  protected _validate(
      date: string, hasDate: boolean, hours: string, minutes: string, isPm: boolean): boolean {
    // Reset error message
    this._errorMessage = '';

    // Unset the datetime if the this date-time-picker is not required as indicated by the hasData
    // checkbox. This element's datetime is then still considered "valid" so that there is a way to
    // indicate that the element is hidden.
    if (!hasDate) {
      this.datetime = undefined;
      return true;
    }

    // Check that date is of form YYYY/MM/DD and is a valid date.
    if (!(date.match(DATE_FORMAT_PATTERN)) || isNaN(Date.parse(date))) {
      this._errorMessage = 'Must be valid date of the form YYYY/MM/DD';
      return false;
    }

    // Ensure time is of correct format
    if (!this._verifyTimeFormat(hours, minutes)) {
      this._errorMessage = 'Must enter valid 12-hour time';
      return false;
    }

    const selectedDateTime = this._toDate(date, hours, minutes, isPm);
    if (!selectedDateTime) {
      this._errorMessage = 'The start date/time is invalid';
      return false;
    }

    // Ensure that start /time is not in the past.
    if (selectedDateTime < new Date()) {
      this._errorMessage = 'Date/time cannot be in the past';
      return false;
    }

    // Set datetime if everything is valid.
    this.datetime = this._toDate(this._date, this._hours, this._minutes, this._isPm);
    return true;
  }

  protected _toDate(date: string, hours: string, minutes: string, isPm: boolean): Date {
    const dateAsArray = date.split('/').map((s) => Number(s));
    // If/when start/end have different backing time arrays, this will need to
    // be updated.
    return new Date(
        dateAsArray[0],
        dateAsArray[1] - 1,
        dateAsArray[2],
        (+hours % 12) + (isPm ? 12 : 0),
        +minutes);
  }

  private _verifyTimeFormat(hours: string, minutes: string): boolean {
    return +hours > 0 && +hours < 13 && +minutes >= 0 && +minutes < 60;
  }
}
