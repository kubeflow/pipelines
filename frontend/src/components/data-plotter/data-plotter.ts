import 'polymer/polymer.html';
import 'polymer/polymer-element.html';

import { customElement, property, observe } from '../../decorators';

import './data-plotter.html';

import * as Apis from '../../lib/apis';
import { select as d3select, csvParseRows } from 'd3';

@customElement
export default class DataPlotter extends Polymer.Element {
  @property({ type: String })
  filePath = '';

  @observe('filePath')
  protected async _filePathChanged(newFilePath: string) {
    if (newFilePath) {
      const data = await Apis.readFile(newFilePath);
      var parsedCSV = csvParseRows(data);

      d3select(this.$.plot)
        .append("table")

        .selectAll("tr")
        .data(parsedCSV).enter()
        .append("tr")

        .selectAll("td")
        .data(function (d) { return d; }).enter()
        .append("td")
        .text(function (d) { return d; });
    }
  }
}
