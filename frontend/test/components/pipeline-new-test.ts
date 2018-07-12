import * as sinon from 'sinon';
import * as assert from '../../node_modules/assert/assert';
import * as APIs from '../../src/lib/apis';

import { Parameter } from '../../src/api/parameter';
import { Pipeline } from '../../src/api/pipeline';
import { PipelinePackage } from '../../src/api/pipeline_package';
import { PipelineNew } from '../../src/components/pipeline-new/pipeline-new';
import { PipelineSchedule } from '../../src/components/pipeline-schedule/pipeline-schedule';
import { resetFixture } from './test-utils';

import * as fixedData from '../../mock-backend/fixed-data';

const packages = fixedData.namedPackages;

const TEST_TAG = 'pipeline-new';

let fixture: PipelineNew;
let getPackagesStub: sinon.SinonStub;

async function _resetFixture(): Promise<void> {
  return resetFixture('pipeline-new', null, (f: PipelineNew) => {
    fixture = f;
    return f.load('', {});
  });
}

describe('pipeline-new', () => {

  before(() => {
    getPackagesStub = sinon.stub(APIs, 'getPackages');
    getPackagesStub.returns({ packages: fixedData.data.packages });
  });

  /** Unset all fields and dropdowns */
  beforeEach(async () => {
    await _resetFixture();
  });

  it('starts out with deploy button disabled', () => {
    assert(fixture.deployButton.disabled, 'Deploy button should initially be disabled.');
  });

  it('starts out with no Pipeline package selected', () => {
    assert.strictEqual(fixture.listBox.selected, -1,
        'No Pipeline package should initially be selected.');
  });

  it('starts out with Pipeline name field empty', () => {
    assert.strictEqual(fixture.nameInput.value, '',
        'Pipeline name field should initially be empty.');
  });

  it('starts out with Pipeline description field empty', () => {
    assert.strictEqual(fixture.descriptionInput.value, '',
        'Pipeline description field should initially be empty.');
  });

  it('starts out with no parameter fields displayed', () => {
    assert(!fixture.$.pipelineParameters.querySelector('#parametersTitle'),
        'Parameters title should not exist before package is selected');
    assert.strictEqual(fixture.$.pipelineParameters.querySelectorAll('paper-input').length, 0,
        'Package parameters should not exist before package is selected');
  });

  it('package selector should contain all packages', () => {
    assert.deepStrictEqual(
        fixture.listBox.items.map((item) => item.packageId),
        fixture.packages.map((pkg) => pkg.id));
  });

  it('displays parameter fields after package is selected', async () => {
    getPackagesStub.returns({ packages: [packages.examplePackage] });
    await _resetFixture();

    fixture.listBox.select(0);
    Polymer.flush();

    assert(fixture.$.pipelineParameters.querySelector('#parametersTitle'),
        'Package parameters should exist after package is selected');
    Array.from(fixture.$.pipelineParameters.querySelectorAll('div.param-name')).forEach(
        (name, i) => {
          assert.strictEqual(
              (name as HTMLElement).innerText,
              packages.examplePackage.parameters[i].name);
        }
    );
  });

  it('updates the displayed parameters when package selection changes', async () => {
    getPackagesStub.returns({ packages: [packages.examplePackage, packages.examplePackage2] });
    await _resetFixture();

    fixture.listBox.select(0);
    Polymer.flush();
    const oldParamNames = JSON.stringify(
        Array.from(fixture.$.pipelineParameters.querySelectorAll('div.param-name'))
            .map((name) => (name as HTMLElement).innerText));

    fixture.listBox.select(1);
    Polymer.flush();
    const newParamNames = JSON.stringify(
        Array.from(fixture.$.pipelineParameters.querySelectorAll('div.param-name'))
            .map((name) => (name as HTMLElement).innerText));

    assert.notStrictEqual(oldParamNames, newParamNames);
  });

  it('clears the displayed parameters when package with no params is selected', async () => {
    getPackagesStub.returns({ packages: [packages.examplePackage, packages.noParamsPackage] });
    await _resetFixture();

    fixture.listBox.select(0);
    Polymer.flush();
    assert(fixture.$.pipelineParameters.querySelectorAll('div.param-name').length > 0,
        'This selected package should have at least 1 parameter.');

    fixture.listBox.select(1);
    Polymer.flush();
    assert(_paramsNotVisible());
  });

  it('clears the displayed parameters when package with undefined params is selected', async () => {
    getPackagesStub.returns(
        { packages: [packages.examplePackage, packages.undefinedParamsPackage] });
    await _resetFixture();

    fixture.listBox.select(0);
    Polymer.flush();
    assert(fixture.$.pipelineParameters.querySelectorAll('div.param-name').length > 0,
        'This selected package should have at least 1 parameter.');

    fixture.listBox.select(1);
    Polymer.flush();
    assert(_paramsNotVisible());
  });

  it('enables deploy button if package selected and Pipeline name is not empty', () => {
    fixture.listBox.select(0);
    Polymer.flush();
    fixture.nameInput.value = 'Some Pipeline Name';

    assert(!fixture.deployButton.disabled,
        'Deploy button should be enabled if package selected and Pipeline name is not empty.');
  });

  it('disables deploy button if no package is selected', () => {
    // Ensure that deploy button is enabled before testing lack of selected package.
    fixture.listBox.select(0);
    Polymer.flush();
    fixture.nameInput.value = 'Some Pipeline Name';
    assert(!fixture.deployButton.disabled, 'Deploy button should be enabled before real test.');

    fixture.listBox.select(-1);
    Polymer.flush();

    assert(fixture.deployButton.disabled,
        'Deploy button should be disabled if no package selected.');
  });

  it('disables deploy button if Pipeline name is empty', () => {
    // Ensure that deploy button is enabled before testing lack of name.
    fixture.listBox.select(0);
    Polymer.flush();
    fixture.nameInput.value = 'Some Pipeline Name';
    assert(!fixture.deployButton.disabled,
        'Deploy button should be enabled before method under test.');

    fixture.nameInput.value = '';

    assert(fixture.deployButton.disabled,
        'Deploy button should be disabled if Pipeline name is empty.');
  });

  it('constructs and passes the correct Pipeline on deploy', () => {
    const deployPipelineStub = sinon.stub(APIs, 'newPipeline');
    deployPipelineStub.returns({});

    fixture.listBox.select(0);
    Polymer.flush();
    fixture.nameInput.value = 'Some Pipeline Name';
    fixture.descriptionInput.value = 'The Pipeline description';
    const parameterInputs = fixture.$.pipelineParameters.querySelectorAll('paper-input');
    parameterInputs.forEach((input, index) => (input as PaperInputElement).value = index + '');
    const schedule = fixture.$.schedule.querySelector('pipeline-schedule') as PipelineSchedule;
    // Now the Pipeline will have a schedule
    (schedule.$.scheduleTypeListbox as PaperListboxElement).select(1);
    Polymer.flush();

    fixture.deployButton.click();

    assert(deployPipelineStub.calledOnce, 'Apis.newPipeline() should only be called once.');
    const actualPipeline = deployPipelineStub.firstCall.args[0];
    assert.strictEqual(actualPipeline.name, fixture.nameInput.value);
    assert.strictEqual(actualPipeline.description, fixture.descriptionInput.value);

    // TODO: mock time and test format.
    assert.strictEqual(actualPipeline.package_id, (fixture.listBox.selectedItem as any).packageId);
    assert.strictEqual(actualPipeline.parameters.length, parameterInputs.length);
    for (let i = 0; i < parameterInputs.length; i++) {
      assert.strictEqual(
          actualPipeline.parameters[i].value, (parameterInputs[i] as PaperInputElement).value);
    }
    // "0 0 * * * ?" is the default schedule. It means "run every hour".
    assert.strictEqual(actualPipeline.trigger.cron_schedule.cron, '0 0 * * * ?');

    deployPipelineStub.restore();
  });

  describe('cloning', () => {

    it('starts out with a Pipeline package selected', async () => {
      getPackagesStub.returns({ packages: [packages.noParamsPackage] });

      await fixture.load('', {}, { packageId: packages.noParamsPackage.id, parameters: [] });
      Polymer.flush();

      assert.strictEqual(
          (fixture.listBox.selectedItem as any).packageId,
          packages.noParamsPackage.id,
          'Pipeline package should initially be selected.');
      assert(_paramsNotVisible());
    });

    it('starts out with empty Pipeline name', async () => {
      getPackagesStub.returns({ packages: [packages.noParamsPackage] });

      await fixture.load('', {}, { packageId: packages.noParamsPackage.id, parameters: [] });
      Polymer.flush();

      assert.strictEqual(fixture.nameInput.value, '',
          'Pipeline name field should initially be empty.');
    });

    it('starts out with Pipeline parameters matching cloned Pipeline', async () => {
      getPackagesStub.returns({ packages: [packages.examplePackage2] });

      await fixture.load('', {}, {
        packageId: packages.examplePackage2.id,
        parameters: [
          { name: 'project', value: 'a-project-name' },
          { name: 'workers', value: '10' },
          { name: 'rounds', value: '5' },
          { name: 'output', value: 'some/output/path' }
        ]
      });
      Polymer.flush();

      const params = Array.from(fixture.$.pipelineParameters.querySelectorAll('paper-input'))
          .map((param) => (param as PaperInputElement).value);
      assert.strictEqual(params[0], 'a-project-name');
      assert.strictEqual(params[1], '10');
      assert.strictEqual(params[2], '5');
      assert.strictEqual(params[3], 'some/output/path');
      Array.from(fixture.$.pipelineParameters.querySelectorAll('div.param-name')).forEach(
          (name, i) => {
            assert.strictEqual(
                (name as HTMLElement).innerText,
                packages.examplePackage2.parameters[i].name);
          }
      );
    });
  });

  after(() => {
    document.body.removeChild(fixture);
    getPackagesStub.restore();
  });
});

function _paramsNotVisible(): boolean {
  const params = fixture.$.pipelineParameters.querySelectorAll('div.param-name');
  return params.length === 0 || Array.from(params)
    .map((el) => _isNotVisible(el as HTMLElement))
    .reduce((prev, cur) => prev && cur);
}

function _isNotVisible(el: HTMLElement): boolean {
  return !el || el.hidden || el.style.visibility === 'hidden' || el.style.display === 'none';
}
