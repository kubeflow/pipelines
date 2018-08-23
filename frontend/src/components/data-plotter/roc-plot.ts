// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import * as d3 from 'd3';

export interface RocOptions {
  data: string[][] | number[][];
  height: number;
  margin: number;
  width: number;
  lineColor: string;
  container: HTMLElement;
}

/**
 * Draws an ROC curve given the data in the following format:
 * [
 *   [FPR, TPR, threshold],
 *   [FPR, TPR, threshold],
 *   ...
 * ]
 */
export function drawROC(config: RocOptions): void {
  const margin = config.margin;
  const width = config.width;
  const height = config.height;
  const lineColor = config.lineColor;
  const svg = d3.select(config.container)
    .append('svg')
    .attr('width', width + 2 * margin)
    .attr('height', height + 2 * margin)
    .append('g')
    .attr('transform', `translate(${margin}, ${margin})`);

  // A dotted reference line for baseline comparison of ROC curve.
  svg.append('line')
    .attr('class', 'dottedLine')
    .attr('x1', 0)
    .attr('y1', height)
    .attr('x2', width)
    .attr('y2', 0);

  const x = d3.scaleLinear().range([0, width]);
  const y = d3.scaleLinear().range([height, 0]);

  const valueline = d3.line()
    .x((d) => x(d[0]))
    .y((d) => y(d[1]));

  svg.append('path')
    .data([config.data as any])
    .attr('class', 'line')
    .attr('d', valueline);

  svg.append('g')
    .attr('transform', `translate(0, ${height})`)
    .call(d3.axisBottom(x) as any)
    .append('text')
    .attr('fill', lineColor)
    .attr('x', width - margin)
    .attr('y', 35)
    .style('font-size', '12px')
    .text('FPR');

  svg.append('g')
    .call(d3.axisLeft(y) as any)
    .append('text')
    .attr('fill', lineColor)
    .attr('transform', 'rotate(-90)')
    .attr('y', -30)
    .style('font-size', '12px')
    .text('TPR');

  const point = svg.append('g')
    .attr('class', 'point')
    .style('display', 'none');

  point.append('circle')
    .attr('r', 4);

  const textOffsetX = 10;
  const hoverText = point.append('text')
    .attr('x', textOffsetX)
    .attr('y', 5);

  const hoverTextTPR = hoverText.append('tspan')
    .attr('dy', 0);

  const hoverTextFPR = hoverText.append('tspan')
    .attr('x', textOffsetX)
    .attr('dy', '1.2em');

  const hoverTextThreshold = hoverText.append('tspan')
    .attr('x', textOffsetX)
    .attr('dy', '1.2em');

  svg.append('rect')
    .attr('class', 'overlay')
    .attr('width', width)
    .attr('height', height)
    .on('mouseover', () => point.style('display', 'initial'))
    .on('mouseout', () => point.style('display', 'none'))
    .on('mousemove', mousemove);

  // Transform the data into an array of x-values (domain) that can be bisected.
  const xs = (config.data as any[][]).map((a) => a[0]);

  // Provides an interactive dot and label that moves along the plot as the
  // mouse moves.
  function mousemove(): void {
    // Convert the mouse's horizontal position into a value in the domain.
    const mouseX = x.invert(d3.mouse(svg.node() as any)[0]);

    // Find the insertion point to the right of the mouse's x-value.
    const insertionPoint = d3.bisectRight(xs, mouseX);

    // Use the data point that's closest to the mouse, not only the left or only
    // the right in order to make the movement feel more natural.
    const dLeft = config.data[insertionPoint - 1] as number[];
    const dRight = config.data[insertionPoint] as number[];
    const d = mouseX - dLeft[0] > dRight[0] - mouseX ? dRight : dLeft;

    point.attr('transform', `translate(${x(d[0])}, ${y(d[1])})`);

    // -Math.max here keeps the label from moving outside of the plot element.
    hoverText.style('transform', `translate(-${Math.max(0, x(d[0]) - width + 100)}px, 12px)`);

    hoverTextTPR.text('TPR: ' + (+(d[1])).toFixed(5));
    hoverTextFPR.text('FPR: ' + (+(d[0])).toFixed(5));
    hoverTextThreshold.text('Threshold: ' + (+(d[2])).toFixed(5));
  }
}
