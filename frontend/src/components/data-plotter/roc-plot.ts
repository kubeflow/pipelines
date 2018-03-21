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
 *   [FPR, TPR],
 *   [FPR, TPR],
 *   ...
 * ]
 */
export function drawROC(config: RocOptions) {
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

  const x = d3.scaleLinear().range([0, width]);
  const y = d3.scaleLinear().range([height, 0]);

  const valueline = d3.line()
    .x((d) => x(d[0]))
    .y((d) => y(d[1]));

  svg.append('path')
    .data([config.data as any])
    .attr('class', 'line')
    .attr('d', valueline);

  const cells = svg.append('g').attr('class', 'vors').selectAll('g');

  const cell = cells.data(config.data as any);
  cell.exit().remove();

  const cellEnter = cell.enter().append('g');

  cellEnter.append('circle')
    .attr('class', 'dot')
    .attr('r', 3.5)
    .attr('cx', (d) => x((d as any)[0]))
    .attr('cy', (d) => y((d as any)[1]));

  cell.select('path').attr('d', (d) => 'M' + (d as any)[2].join('L') + 'Z');

  cellEnter.append('text').attr('class', 'hidetext')
    .attr('x', (d) => Math.min(width - 100, x((d as any)[0]))) // Don't go over width
    .attr('y', (d) => y((d as any)[1]) + 20)
    .text((d) => 'threshold: ' + (+(d as any)[2]).toFixed(5));

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
}
