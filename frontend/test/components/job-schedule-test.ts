import * as assert from '../../node_modules/assert/assert';
import * as Utils from '../../src/lib/utils';

import { CronSchedule, Job, PeriodicSchedule, Trigger } from '../../src/api/job';
import { JobSchedule } from '../../src/components/job-schedule/job-schedule';
import { dialogStub, isVisible, notificationStub, resetFixture } from './test-utils';

let fixture: JobSchedule;

async function _resetFixture(): Promise<void> {
  return resetFixture('job-schedule', (f: JobSchedule) => {
    fixture = f;
  });
}

describe('job-schedule', () => {

  beforeEach(async () => {
    await _resetFixture();
  });

  it('defaults to schedule type: "Run right away"', () => {
    assert.strictEqual(fixture.scheduleTypeDropdown.value, 'Run right away');
  });

  it('returns null when toTrigger is called with no schedule', () => {
    assert.strictEqual(fixture.toTrigger(), null);
  });

  it('does not include "has start date" checkbox by default with no schedule', () => {
    assert.strictEqual(fixture.startDateTimePicker, null);
  });

  it('does not include "has end date" checkbox by default with no schedule', () => {
    assert.strictEqual(fixture.endDateTimePicker, null);
  });

  it('does not include interval dropdown by default with no schedule', () => {
    assert.strictEqual(fixture.cronIntervalDropdown, null);
  });

  describe('periodic schedule-specific ui', () => {

    beforeEach(() => {
      // Select periodic schedule type
      fixture.scheduleTypeListbox.select(1);
      assert.strictEqual(fixture.scheduleTypeDropdown.value, 'Periodic');

      // Ensure that Polymer updates before inspecting elements
      Polymer.flush();
    });

    it('shows start/end date/time checkboxes unchecked by default', () => {
      assert.strictEqual(fixture.startDateTimePicker.useDateTimeCheckbox.checked, false,
          '"Has start date" checkbox should be unchecked by default');
      assert.strictEqual(fixture.endDateTimePicker.useDateTimeCheckbox.checked, false,
          '"Has end date" checkbox should be unchecked by default');
    });

    it('shows interval dropdown with "hours" selected by default', (done) => {
      assert.notStrictEqual(
          fixture.periodicIntervalDropdown, null, 'interval dropdown should not be null');
      // Wrapper here because the element doesn't immediately render otherwise,
      // even with Polymer.flush()
      Polymer.Async.idlePeriod.run(() => {
        assert.strictEqual(fixture.periodicIntervalDropdown.value, 'hours');
        // Default interval is 1 hour (3600 seconds)
        assert.deepStrictEqual(fixture.toTrigger(), new Trigger(new PeriodicSchedule(3600)));
        done();
      });
    });

    it('updates the seconds in a period when the recurrence interval is changed', (done) => {
      const periods = [
        60, // 'minute'
        60 * 60, // 'hour'
        60 * 60 * 24, // 'day'
        60 * 60 * 24 * 7, // 'week'
        60 * 60 * 24 * 30, // 'month'
      ];
      const frequency = 3;
      Polymer.Async.idlePeriod.run(() => {
        fixture.periodFrequencyInput.value = frequency + '';
        fixture._periodicIntervals.forEach((v, i) => {
          fixture.periodicIntervalListbox.select(i);
          assert.strictEqual(fixture.periodicIntervalDropdown.value, v);
          assert.deepStrictEqual(
              fixture.toTrigger(), new Trigger(new PeriodicSchedule(periods[i] * frequency)));
        });
        done();
      });
    });

    it('returns a periodic_schedule trigger when toTrigger is called', () => {
      // Default periodic schedule corresponds to "every hour", so 3600 seconds
      const expectedPeriodicTrigger = new Trigger(new PeriodicSchedule(3600));
      assert.deepStrictEqual(fixture.toTrigger(), expectedPeriodicTrigger);
    });

  });

  describe('cron-specific ui', () => {

    beforeEach(() => {
      // Select cron schedule type
      fixture.scheduleTypeListbox.select(2);
      assert.strictEqual(fixture.scheduleTypeDropdown.value, 'Cron');

      // Ensure that Polymer updates before inspecting elements
      Polymer.flush();
    });

    it('shows start/end date/time checkboxes unchecked by default', () => {
      assert.strictEqual(fixture.startDateTimePicker.useDateTimeCheckbox.checked, false,
          '"Has start date" checkbox should be unchecked by default');
      assert.strictEqual(fixture.endDateTimePicker.useDateTimeCheckbox.checked, false,
          '"Has end date" checkbox should be unchecked by default');
    });

    it('shows interval dropdown with "hourly" selected by default', (done) => {
      assert.notStrictEqual(
          fixture.cronIntervalDropdown, null, 'interval dropdown should not be null');
      // Wrapper here because the element doesn't immediately render otherwise,
      // even with Polymer.flush()
      Polymer.Async.idlePeriod.run(() => {
        assert.strictEqual(fixture.cronIntervalDropdown.value, 'hourly');
        assert.deepStrictEqual(fixture.toTrigger(), new Trigger(new CronSchedule('0 0 * * * ?')));
        done();
      });
    });

    it('shows "all weekdays" checkbox checked and disabled by default', () => {
      assert.strictEqual(fixture.allWeekdaysCheckbox.checked, true);
      assert.strictEqual(fixture.allWeekdaysCheckbox.disabled, true);
    });

    it('updates the cron expression when the recurrence interval is changed', (done) => {
      const cronExpressions = [
        '0 * * * * ?', // 'every minute'
        '0 0 * * * ?', // 'hourly'
        '0 0 0 * * ?', // 'daily'
        '0 0 0 ? * *', // 'weekly'
        '0 0 0 0 * ?', // 'monthly'
      ];
      Polymer.Async.idlePeriod.run(() => {
        fixture._cronIntervals.forEach((v, i) => {
          fixture.cronIntervalListbox.select(i);
          assert.strictEqual(fixture.cronIntervalDropdown.value, v);
          assert.deepStrictEqual(
              fixture.toTrigger(),
              new Trigger(new CronSchedule(cronExpressions[i])));
          assert.strictEqual(fixture.allWeekdaysCheckbox.checked, true);
          assert.strictEqual(fixture.allWeekdaysCheckbox.disabled, v !== 'weekly');
        });
        done();
      });
    });

    it('enables the weekday checkbox and buttons when the interval is "weekly"', (done) => {
      Polymer.Async.idlePeriod.run(() => {
        // Set interval to "weekly"
        fixture.cronIntervalListbox.select(3);
        assert.strictEqual(fixture.cronIntervalDropdown.value, 'weekly');
        assert.strictEqual(fixture.allWeekdaysCheckbox.checked, true);
        assert.strictEqual(fixture.allWeekdaysCheckbox.disabled, false);
        const sundayButton =
        fixture.shadowRoot.querySelector('#weekdayButtons paper-button') as PaperButtonElement;
        sundayButton.click();
        assert.strictEqual(sundayButton.active, false);
        // 1-6 correspond to Monday - Saturday, Sunday isn't included because we clicked it.
        assert.deepStrictEqual(
            fixture.toTrigger(), new Trigger(new CronSchedule('0 0 0 ? * 1,2,3,4,5,6')));
        done();
      });
    });

    it('returns a cron_schedule trigger when toTrigger is called', () => {
      // Default cron schedule corresponds to "hourly"
      const expectedCronTrigger = new Trigger(new CronSchedule('0 0 * * * ?'));
      assert.deepStrictEqual(fixture.toTrigger(), expectedCronTrigger);
    });

    it('returns manually entered cron expression when toTrigger is called', () => {
      // We use the CSS class rather than the element property because of an issue with Polymer
      // inputs and tabindex. See: https://github.com/PolymerElements/iron-behaviors/pull/83
      assert.strictEqual(fixture.cronExpressionInput.classList.contains('disabled'), true);

      // Because we set the cron expression directly, checking this checkbox isn't actually
      // necessary. Thus we just check that the input field went from disabled to enabled.
      fixture.allowEditingCronCheckbox.checked = true;
      fixture.cronExpressionInput.value = '1 2 3 4 5 ?';

      assert.strictEqual(fixture.cronExpressionInput.classList.contains('disabled'), false);
      const expectedCronTrigger = new Trigger(new CronSchedule('1 2 3 4 5 ?'));
      assert.deepStrictEqual(fixture.toTrigger(), expectedCronTrigger);
    });

    it('rederives cron expression from inputs when editing is redisabled', () => {
      // Default cron schedule corresponds to "hourly"
      const originalCronTrigger = new Trigger(new CronSchedule('0 0 * * * ?'));
      assert.deepStrictEqual(fixture.toTrigger(), originalCronTrigger);
      assert.strictEqual(fixture.cronExpressionInput.classList.contains('disabled'), true);

      // Enable editing and change the cron expression
      fixture.allowEditingCronCheckbox.checked = true;
      fixture.cronExpressionInput.value = '1 2 3 4 5 ?';
      const newCronTrigger = new Trigger(new CronSchedule('1 2 3 4 5 ?'));
      assert.deepStrictEqual(fixture.toTrigger(), newCronTrigger);
      assert.strictEqual(fixture.cronExpressionInput.classList.contains('disabled'), false);

      // Redisable editing of cron expression. Cron expression should revert to original
      fixture.allowEditingCronCheckbox.checked = false;
      assert.deepStrictEqual(fixture.toTrigger(), originalCronTrigger);
      assert.strictEqual(fixture.cronExpressionInput.classList.contains('disabled'), true);
    });

    it('allows/disallows tabbing to the cron expression field depending on disabled state', () => {
      // Should initially forbid tabbing as the field is disabled by default
      assert.strictEqual(fixture.cronExpressionInput.classList.contains('disabled'), true);
      assert.strictEqual(fixture.cronExpressionInput.getAttribute('tabindex'), '-1');

      // Should allow tabbing when the cron field is editable
      fixture.allowEditingCronCheckbox.checked = true;
      assert.strictEqual(fixture.cronExpressionInput.classList.contains('disabled'), false);
      assert.strictEqual(fixture.cronExpressionInput.getAttribute('tabindex'), '0');

      // Should not allow tabbing when the cron field is once again disabled
      fixture.allowEditingCronCheckbox.checked = false;
      assert.strictEqual(fixture.cronExpressionInput.classList.contains('disabled'), true);
      assert.strictEqual(fixture.cronExpressionInput.getAttribute('tabindex'), '-1');
    });

  });

  after(() => {
    document.body.removeChild(fixture);
  });

});
