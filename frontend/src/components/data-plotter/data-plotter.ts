import 'polymer/polymer-element.html';
import 'polymer/polymer.html';

import { customElement, observe, property } from '../../decorators';

import './data-plotter.html';

import * as Apis from '../../lib/apis';

import { csvParseRows, select as d3select } from 'd3';

@customElement
export class DataPlotter extends Polymer.Element {
  @property({ type: String })
  filePath = '';

  @observe('filePath')
  protected async _filePathChanged(newFilePath: string) {
    if (newFilePath) {
      const data = await Apis.readFile(newFilePath);
      const parsedCSV = csvParseRows(data);

      d3select(this.$.plot)
        .append('table')

        .selectAll('tr')
        .data(parsedCSV).enter()
        .append('tr')

        .selectAll('td')
        .data((d) => d).enter()
        .append('td')
        .text((d) => d);
    }
  }
}
