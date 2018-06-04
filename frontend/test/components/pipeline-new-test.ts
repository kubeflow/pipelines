import * as assert from 'assert';
import * as sinon from 'sinon';
import * as APIs from '../../src/lib/apis';

import { Parameter } from '../../src/api/parameter';
import { Pipeline } from '../../src/api/pipeline';
import { PipelinePackage } from '../../src/api/pipeline_package';
import { PipelineNew } from '../../src/components/pipeline-new/pipeline-new';
import { PipelineSchedule } from '../../src/components/pipeline-schedule/pipeline-schedule';

import * as fixedData from '../../mock-backend/fixed-data';

const packages = fixedData.namedPackages;

const TEST_TAG = 'pipeline-new';

let fixture: PipelineNew;
let getPackagesStub: sinon.SinonStub;
let newPipelineSpy: sinon.SinonSpy;

// Aliases for commonly used elements
let packagesListbox: PaperListboxElement;
let nameInput: PaperInputElement;
let descriptionInput: PaperInputElement;
let deployButton: PaperButtonElement;

// TODO: Refactor the common parts of the resetFixture functions into a util file.
async function resetFixture(): Promise<void> {
  const old = document.querySelector(TEST_TAG);
  if (old) {
    document.body.removeChild(old);
  }
  document.body.appendChild(document.createElement(TEST_TAG));
  fixture = document.querySelector(TEST_TAG) as PipelineNew;
  await fixture.load('', {});

  Polymer.flush();

  packagesListbox = fixture.$.packagesListbox as PaperListboxElement;
  nameInput = fixture.$.name as PaperInputElement;
  descriptionInput = fixture.$.description as PaperInputElement;
  deployButton = fixture.$.deployButton as PaperButtonElement;
}

describe('pipeline-new', () => {

  before(() => {
    getPackagesStub = sinon.stub(APIs, 'getPackages');
    getPackagesStub.returns({ packages: fixedData.data.packages });
    newPipelineSpy = sinon.spy(APIs, 'newPipeline');
  });

  /** Unset all fields and dropdowns */
  beforeEach(async () => {
    await resetFixture();
  });

  it('starts out with deploy button disabled', () => {
    assert(deployButton.disabled, 'Deploy button should initially be disabled.');
  });

  it('starts out with no Pipeline package selected', () => {
    assert.strictEqual(packagesListbox.selected, -1,
        'No Pipeline package should initially be selected.');
  });

  it('starts out with Pipeline name field empty', () => {
    assert.strictEqual(nameInput.value, '', 'Pipeline name field should initially be empty.');
  });

  it('starts out with Pipeline description field empty', () => {
    assert.strictEqual(descriptionInput.value, '',
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
        packagesListbox.items.map((item) => item.packageId),
        fixture.packages.map((pkg) => pkg.id));
  });

  it('displays parameter fields after package is selected', async () => {
    getPackagesStub.returns({ packages: [packages.examplePackage] });
    await resetFixture();

    packagesListbox.select(0);
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
    await resetFixture();

    packagesListbox.select(0);
    Polymer.flush();
    const oldParamNames = JSON.stringify(
        Array.from(fixture.$.pipelineParameters.querySelectorAll('div.param-name'))
            .map((name) => (name as HTMLElement).innerText));

    packagesListbox.select(1);
    Polymer.flush();
    const newParamNames = JSON.stringify(
        Array.from(fixture.$.pipelineParameters.querySelectorAll('div.param-name'))
            .map((name) => (name as HTMLElement).innerText));

    assert.notStrictEqual(oldParamNames, newParamNames);
  });

  it('clears the displayed parameters when package with no params is selected', async () => {
    getPackagesStub.returns({ packages: [packages.examplePackage, packages.noParamsPackage] });
    await resetFixture();

    packagesListbox.select(0);
    Polymer.flush();
    assert(fixture.$.pipelineParameters.querySelectorAll('div.param-name').length > 0,
        'This selected package should have at least 1 parameter.');

    packagesListbox.select(1);
    Polymer.flush();
    assert(_paramsNotVisible());
  });

  it('clears the displayed parameters when package with undefined params is selected', async () => {
    getPackagesStub.returns(
        { packages: [packages.examplePackage, packages.undefinedParamsPackage] });
    await resetFixture();

    packagesListbox.select(0);
    Polymer.flush();
    assert(fixture.$.pipelineParameters.querySelectorAll('div.param-name').length > 0,
        'This selected package should have at least 1 parameter.');

    packagesListbox.select(1);
    Polymer.flush();
    assert(_paramsNotVisible());
  });

  it('enables deploy button if package selected and Pipeline name is not empty', () => {
    packagesListbox.select(0);
    Polymer.flush();
    nameInput.value = 'Some Pipeline Name';

    assert(!deployButton.disabled,
        'Deploy button should be enabled if package selected and Pipeline name is not empty.');
  });

  it('disables deploy button if no package is selected', () => {
    // Ensure that deploy button is enabled before testing lack of selected package.
    packagesListbox.select(0);
    Polymer.flush();
    nameInput.value = 'Some Pipeline Name';
    assert(!deployButton.disabled, 'Deploy button should be enabled before real test.');

    packagesListbox.select(-1);
    Polymer.flush();

    assert(deployButton.disabled, 'Deploy button should be disabled if no package selected.');
  });

  it('disables deploy button if Pipeline name is empty', () => {
    // Ensure that deploy button is enabled before testing lack of name.
    packagesListbox.select(0);
    Polymer.flush();
    nameInput.value = 'Some Pipeline Name';
    assert(!deployButton.disabled, 'Deploy button should be enabled before method under test.');

    nameInput.value = '';

    assert(deployButton.disabled, 'Deploy button should be disabled if Pipeline name is empty.');
  });

  it('constructs and passes the correct Pipeline on deploy', () => {
    packagesListbox.select(0);
    Polymer.flush();
    nameInput.value = 'Some Pipeline Name';
    descriptionInput.value = 'The Pipeline description';
    const parameterInputs = fixture.$.pipelineParameters.querySelectorAll('paper-input');
    parameterInputs.forEach((input, index) => (input as PaperInputElement).value = index + '');
    const schedule = fixture.$.schedule.querySelector('pipeline-schedule') as PipelineSchedule;
    // Now the Pipeline will have a schedule
    (schedule.$.scheduleTypeListbox as PaperListboxElement).select(1);
    Polymer.flush();

    deployButton.click();

    assert(newPipelineSpy.calledOnce, 'Apis.newPipeline() should only be called once.');
    const actualPipeline = newPipelineSpy.firstCall.args[0];
    assert.strictEqual(actualPipeline.name, nameInput.value);
    assert.strictEqual(actualPipeline.description, descriptionInput.value);
    // TODO: mock time and test format.
    assert(actualPipeline.created_at, 'Deployed Pipeline should have created_at property set.');
    assert.strictEqual(actualPipeline.package_id, (packagesListbox.selectedItem as any).packageId);
    assert.strictEqual(actualPipeline.parameters.length, parameterInputs.length);
    for (let i = 0; i < parameterInputs.length; i++) {
      assert.strictEqual(
          actualPipeline.parameters[i].value, (parameterInputs[i] as PaperInputElement).value);
    }
    // "0 0 * * * ?" is the default schedule. It means "run every hour".
    assert.strictEqual(actualPipeline.schedule, '0 0 * * * ?');
  });

  describe('cloning', () => {

    it('starts out with a Pipeline package selected', async () => {
      getPackagesStub.returns({ packages: [packages.noParamsPackage] });

      await fixture.load('', {}, { packageId: packages.noParamsPackage.id, parameters: [] });
      Polymer.flush();

      assert.strictEqual(
          (packagesListbox.selectedItem as any).packageId,
          packages.noParamsPackage.id,
          'Pipeline package should initially be selected.');
      assert(_paramsNotVisible());
    });

    it('starts out with empty Pipeline name', async () => {
      getPackagesStub.returns({ packages: [packages.noParamsPackage] });

      await fixture.load('', {}, { packageId: packages.noParamsPackage.id, parameters: [] });
      Polymer.flush();

      assert.strictEqual(nameInput.value, '', 'Pipeline name field should initially be empty.');
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
    newPipelineSpy.restore();
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
