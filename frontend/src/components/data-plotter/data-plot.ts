import { customElement, property } from 'polymer-decorators/src/decorators';
import 'polymer/polymer-element.html';
import 'polymer/polymer.html';

import { csvParseRows } from 'd3';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { PlotMetadata, PlotType } from '../../model/output_metadata';

import { drawMatrix } from './confusion-matrix';
import { drawROC } from './roc-plot';

import './data-plot.html';

@customElement('data-plot')
export class DataPlot extends Polymer.Element {

  @property({ type: Object })
  public plotMetadata: PlotMetadata | null = null;

  @property({ type: String })
  public plotTitle = '';

  ready() {
    super.ready();
    if (this.plotMetadata) {
      switch (this.plotMetadata.type) {
        case PlotType.CONFUSION_MATRIX:
          this._plotConfusionMatrix(this.plotMetadata);
          break;
        case PlotType.ROC:
          this._plotRocCurve(this.plotMetadata);
          break;
        default:
          Utils.log.error('Unknown plotType:', this.plotMetadata.type);
      }
    }
  }

  private async _plotConfusionMatrix(metadata: PlotMetadata) {
    this.plotTitle = 'Confusion Matrix from file: ' + metadata.source;

    const data = csvParseRows(await Apis.readFile(metadata.source));
    const labels = metadata.labels;
    const labelIndex: { [label: string]: number } = {};
    let index = 0;
    labels.forEach((l) => {
      labelIndex[l] = index++;
    });

    const matrix = Array.from(Array(labels.length), () => new Array(labels.length));
    data.forEach(([target, predicted, count]) => {
      const i = labelIndex[target];
      const j = labelIndex[predicted];
      matrix[i][j] = Number.parseInt(count);
    });

    // Render the confusion matrix
    drawMatrix({
      container: this.$.plot as HTMLElement,
      data: matrix,
      endColor: getComputedStyle(this).getPropertyValue('--accent-color'),
      labels,
      startColor: getComputedStyle(this).getPropertyValue('--bg-color')
    });
  }

  private async _plotRocCurve(metadata: PlotMetadata) {
    this.plotTitle = 'ROC curve from file: ' + metadata.source;

    // Render the ROC plot
    drawROC({
      container: this.$.plot as HTMLElement,
      data: csvParseRows(await Apis.readFile(metadata.source)),
      height: 350,
      lineColor: getComputedStyle(this).getPropertyValue('--accent-color'),
      margin: 50,
      width: 550
    });
  }
}
