import * as d3 from 'd3';

export interface RocOptions {
  data: string[][] | number[][];
  height: number;
  margin: number;
  width: number;
}

/**
 * Draws an ROC curve given the data in the following format:
 * [
 *   [FPR, TPR],
 *   [FPR, TPR],
 *   ...
 * ]
 */
export function drawROC(container: HTMLElement, config: RocOptions) {
  const margin = config.margin;
  const width = config.width;
  const height = config.height;
  const svg = d3.select(container)
    .append('svg')
    .attr('width', width + 2 * margin)
    .attr('height', height + 2 * margin)
    .append('g')
    .attr('transform', 'translate(' + margin + ',' + margin + ')');

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
    .attr('transform', 'translate(0,' + height + ')')
    .call(d3.axisBottom(x) as any)
    .append('text')
    .attr('fill', '#000')
    .attr('x', width - margin)
    .attr('y', 35)
    .style('font-size', '12px')
    .text('FPR');

  svg.append('g')
    .call(d3.axisLeft(y) as any)
    .append('text')
    .attr('fill', '#000')
    .attr('transform', 'rotate(-90)')
    .attr('y', -30)
    .style('font-size', '12px')
    .text('TPR');
}
