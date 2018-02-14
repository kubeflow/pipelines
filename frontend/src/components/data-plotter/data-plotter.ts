import 'polymer/polymer-element.html';
import 'polymer/polymer.html';

import { drawMatrix, drawStatsTable } from './confusion_matrix';
import { drawROC } from './roc_plot';

export enum PLOT_TYPE {
  CONFUSION_MATRIX,
}

export class DataPlotter {
  private _plotElement: HTMLElement;

  constructor(plotElement: HTMLElement) {
    this._plotElement = plotElement;
  }

  public plotConfusionMatrix(data: number[][], labels: string[], accentColor: string) {
    // Render the statistics table
    drawStatsTable(this._plotElement, data);

    // Render the matrix
    drawMatrix({
      container: this._plotElement,
      data,
      endColor: accentColor,
      labels,
      startColor: '#ffffff',
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
