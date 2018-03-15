import 'polymer/polymer-element.html';
import 'polymer/polymer.html';

import { drawMatrix } from './confusion-matrix';
import { drawROC } from './roc-plot';

export class DataPlotter {
  private _plotElement: HTMLElement;

  constructor(plotElement: HTMLElement) {
    this._plotElement = plotElement;
  }

  public plotConfusionMatrix(data: number[][], labels: string[],
                             startColor: string, endColor: string) {

    // Render the matrix
    drawMatrix({
      container: this._plotElement,
      data,
      endColor,
      labels,
      startColor,
    });
  }

  public plotRocCurve(data: string[][], lineColor = '#000') {
    const rocChartOptions = {
      data,
      height: 350,
      lineColor,
      margin: 50,
      width: 550,
    };
    drawROC(this._plotElement, rocChartOptions);
  }
}
