import * as assert from '../../node_modules/assert/assert';
import * as Utils from '../../src/lib/utils';

import { CronSchedule, PeriodicSchedule, Pipeline, Trigger } from '../../src/api/pipeline';
import { PipelineSchedule } from '../../src/components/pipeline-schedule/pipeline-schedule';
import { dialogStub, isVisible, notificationStub, resetFixture } from './test-utils';

let fixture: PipelineSchedule;

async function _resetFixture(): Promise<void> {
  return resetFixture('pipeline-schedule', (f: PipelineSchedule) => {
    fixture = f;
  });
}

describe('pipeline-schedule', () => {

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

    it('updates the crontab when the recurrence interval is changed', (done) => {
      const crontabs = [
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
          assert.deepStrictEqual(fixture.toTrigger(), new Trigger(new CronSchedule(crontabs[i])));
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

  });

  after(() => {
    document.body.removeChild(fixture);
  });

});
