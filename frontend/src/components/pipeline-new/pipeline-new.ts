import 'iron-icons/iron-icons.html';
import 'neon-animation/web-animations.html';
import 'paper-checkbox/paper-checkbox.html';
import 'paper-dropdown-menu/paper-dropdown-menu.html';
import 'paper-input/paper-input.html';
import 'paper-item/paper-item.html';
import 'paper-listbox/paper-listbox.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { Parameter } from '../../model/parameter';
import { Pipeline } from '../../model/pipeline';
import { PipelinePackage } from '../../model/pipeline_package';

import './pipeline-new.html';

interface NewPipelineQueryParams {
  packageId?: number;
}

interface NewPipelineData {
  packageId: number;
  parameters: Parameter[];
}

@customElement('pipeline-new')
export class PipelineNew extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public packages: PipelinePackage[];

  @property({ type: Object })
  public newPipeline: Pipeline;

  @property({ type: Number })
  protected _packageIndex = -1;

  @property({ computed: '_updateDeployButtonState(newPipeline.name)', type: Boolean })
  protected inputIsInvalid = false;

  protected _busy = false;
  protected _overwriteData?: NewPipelineData;

  public async load(_: string, queryParams: NewPipelineQueryParams,
                    pipelineData?: NewPipelineData) {
    this._busy = true;
    this._overwriteData = pipelineData;
    const packageList = this.$.packagesListbox as any;

    // Clear package selection on each component load
    packageList.select();
    this.newPipeline = new Pipeline();

    // Initialize input to valid to avoid error messages on page load.
    (this.$.name as any).invalid = false;

    this.set('newPipeline.packageId',
      this._overwriteData ? this._overwriteData.packageId : queryParams.packageId || -1);

    try {
      this.packages = await Apis.getPackages();

      if (this.newPipeline.packageId > -1) {
        this.packages.forEach((p, i) => {
          if (p.id === +this.newPipeline.packageId) {
            // This will cause the observer below to fire before continuing to
            // overwrite the data below.
            this.set('_packageIndex', i);
          }
        });
      }
      if (this._overwriteData) {
        this.set('newPipeline.packageId', this._overwriteData.packageId);
        // Augment the list of parameters with the overwrite data parameters. To
        // achieve this, first deep clone the parameters array, then for each
        // parameter, check if there one with the same name in the overwrite
        // data, Object.assign them.
        const augmentedParams = this.newPipeline.parameters.map((p) => ({...p}));
        this._overwriteData.parameters.forEach((p) => {
          const param = augmentedParams.filter((_p) => _p.name === p.name);
          if (param.length === 1) {
            param[0] = Object.assign(param[0], p);
          }
        });
        this.set('newPipeline.parameters', augmentedParams);
      }
    } finally {
      this._busy = false;
    }
  }

  protected _updateDeployButtonState(pipelineName: string) {
    return !pipelineName;
  }

  @observe('_packageIndex')
  protected _packageIndexChanged(newIndex: number) {
    if (newIndex === undefined || this.packages === undefined) {
      return;
    }
    const pkg = this.packages[newIndex];
    this.set('newPipeline.packageId', pkg.id);
    this.set('newPipeline.parameters', pkg.parameters);
  }

  protected _altUpload() {
    (this.$.altFileUpload as HTMLInputElement).click();
  }

  protected async _upload() {
    const files = (this.$.altFileUpload as HTMLInputElement).files;

    if (!files) {
      return;
    }

    const file = files[0];
    this._busy = true;
    const pkg = await Apis.uploadPackage(file);
    // Add the parsed package to the dropdown list, and select it
    this.push('packages', pkg);
    (this.$.packagesListbox as any).selected = (this.$.packagesListbox as any).items.length;
    this._busy = false;

    (this.$.altFileUpload as HTMLInputElement).value = '';
  }

  protected async _deploy() {
    await Apis.newPipeline(this.newPipeline);
    this.dispatchEvent(new RouteEvent('/pipelines'));
  }
}
