import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'polymer/polymer.html';

import { customElement, property } from '../../decorators';
import { PageElement } from '../../lib/page_element';
import { PipelinePackage } from '../../lib/pipeline_package';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import './package-details.html';

@customElement
export class PackageDetails extends Polymer.Element implements PageElement {

  @property({ type: Object })
  public package: PipelinePackage | null = null;

  public async refresh(path: string) {
    if (path !== '') {
      const id = Number.parseInt(path);
      if (isNaN(id)) {
        Utils.log.error(`Bad package path: ${id}`);
        return;
      }
      this.package = await Apis.getPackage(id);
    }
  }
}
