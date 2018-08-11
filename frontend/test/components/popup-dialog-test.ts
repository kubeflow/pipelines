import * as assert from '../../node_modules/assert/assert';

import { DialogResult, PopupDialog } from '../../src/components/popup-dialog/popup-dialog';
import { isVisible, resetFixture } from './test-utils';

let fixture: PopupDialog;

async function _resetFixture(): Promise<void> {
  return resetFixture('popup-dialog', null, (f: PopupDialog) => {
    fixture = f;
  });
}

describe('popup-dialog', () => {

  beforeEach(async () => {
    await _resetFixture();
  });

  it('shows the title and body', (done) => {
    const title = 'test title';
    const body = 'test body';
    fixture.title = title;
    fixture.body = body;
    fixture.open();
    Polymer.Async.idlePeriod.run(() => {
      assert(isVisible(fixture.titleElement));
      assert.strictEqual(fixture.titleElement.innerText, title);

      assert(isVisible(fixture.bodyElement));
      assert.strictEqual(fixture.bodyElement.innerText, body);
      done();
    });
  });

  it('does not show any buttons if none were specified', (done) => {
    fixture.open();
    Polymer.Async.idlePeriod.run(() => {
      assert(!isVisible(fixture.button1Element));
      assert(!isVisible(fixture.button2Element));
      done();
    });
  });

  it('hides second button if only one was specified', (done) => {
    fixture.button1 = 'Close';
    fixture.open();
    Polymer.Async.idlePeriod.run(() => {
      assert(isVisible(fixture.button1Element));
      // Paper buttons use all caps
      assert.strictEqual(fixture.button1Element.innerText, 'CLOSE');

      assert(!isVisible(fixture.button2Element));
      done();
    });
  });

  it('shows both buttons if two buttons were specified', (done) => {
    fixture.button1 = 'Close';
    fixture.button2 = 'Cancel';
    fixture.open();
    Polymer.Async.idlePeriod.run(() => {
      assert(isVisible(fixture.button1Element));
      // Paper buttons use all caps
      assert.strictEqual(fixture.button1Element.innerText, 'CLOSE');

      assert(isVisible(fixture.button2Element));
      assert.strictEqual(fixture.button2Element.innerText, 'CANCEL');
      done();
    });
  });

  it('clicking outside the dialog returns DISMISS result', (done) => {
    fixture.open()
      .then((result) => {
        assert.strictEqual(result, DialogResult.DISMISS);
        done();
      })
      .catch((e) => {
        done(e);
      });

    Polymer.Async.idlePeriod.run(() => {
      document.body.click();
    });
  });

  it('clicking the first button returns BUTTON1 result', (done) => {
    fixture.button1 = 'Close';
    fixture.open()
      .then((result) => {
        assert.strictEqual(result, DialogResult.BUTTON1);
        done();
      })
      .catch((e) => {
        done(e);
      });

    Polymer.Async.idlePeriod.run(() => {
      fixture.button1Element.click();
    });
  });

  it('clicking the second button returns BUTTON2 result', (done) => {
    fixture.button1 = 'Close';
    fixture.button2 = 'Cancel';
    fixture.open()
      .then((result) => {
        assert.strictEqual(result, DialogResult.BUTTON2);
        done();
      })
      .catch((e) => {
        done(e);
      });

    Polymer.Async.idlePeriod.run(() => {
      fixture.button2Element.click();
    });
  });

  after(() => {
    document.body.removeChild(fixture);
  });

});
