import * as d3 from 'd3';

export interface MatrixOptions {
  data: number[][];
  container: HTMLElement;
  labels: string[];
  startColor: string;
  endColor: string;
  xAxisLabel: string;
  yAxisLabel: string;
}

const margin = { top: 50, right: 50, bottom: 200, left: 200 };

/**
 * Draws a confusion matrix with the given options. One of the options
 * is the HTMLElement container, which gets populated with the matrix
 * elements.
 */
export function drawMatrix(options: MatrixOptions) {
  const width = 750;
  const height = 750;
  const data = options.data;
  const container = options.container;
  const labelsData = options.labels;
  const startColor = options.startColor;
  const endColor = options.endColor;

  const widthLegend = 100;

  const maxValue = d3.max(data, (layer) => d3.max(layer, (d) => d)) as number;
  const minValue = d3.min(data, (layer) => d3.min(layer, (d) => d)) as number;

  const numrows = data.length;
  const numcols = data[0].length;

  const svg = d3.select(container).append('svg')
    .attr('width', width + margin.left + margin.right + widthLegend)
    .attr('height', height + margin.top + margin.bottom)
    .append('g')
    .attr('transform', `translate(${margin.left}, ${margin.top})`);

  svg.append('rect')
    .style('stroke', 'black')
    .style('stroke-width', '2px')
    .attr('width', width)
    .attr('height', height);

  const x = d3.scaleBand()
    .domain(d3.range(numcols).map((c) => c.toString()))
    .range([0, width]);

  const y = d3.scaleBand()
    .domain(d3.range(numrows).map((r) => r.toString()))
    .range([0, height]);

  const colorMap = d3.scaleLinear<string>()
    .domain([minValue, maxValue])
    .range([startColor, endColor]);

  const row = svg.selectAll('.row')
    .data(data)
    .enter().append('g')
      .attr('class', 'row')
      .attr('row', (_, i) => i)
      .attr('transform', (d, i) => `translate(0, ${y(i.toString())})`);

  const cell = row.selectAll('.cell')
    .data((d) => d)
    .enter().append('g')
      .on('mouseover', function() { cellMouse(container, this, true); })
      .on('mouseout', function() { cellMouse(container, this, false); })
      .attr('class', 'cell')
      .attr('column', (_, i) => i)
      .attr('transform', (d, i) => `translate(${x(i.toString())}, 0)`);

  cell.append('rect')
    .attr('width', x.bandwidth())
    .attr('height', y.bandwidth())
    .style('stroke', '#ddd')
    .style('stroke-width', 1);

  cell.append('text')
    .attr('dy', '.32em')
    .attr('x', x.bandwidth() / 2)
    .attr('y', y.bandwidth() / 2)
    .attr('text-anchor', 'middle')
    .style('fill', (d, i) => d >= maxValue / 2 ? 'white' : 'black')
    .text((d, i) => d);

  row.selectAll('.cell')
    .data((d, i) => data[i])
    .style('fill', colorMap);

  const labels = svg.append('g')
    .attr('class', 'labels');

  const columnLabels = labels.selectAll('.column-label')
    .data(labelsData)
    .enter().append('g')
      .attr('class', (_, i) => 'column-label column-' + i)
      .attr('transform', (d, i) => `translate(${x(i.toString())}, ${20 + height})`);

  columnLabels.append('line')
    .style('stroke', 'black')
    .style('stroke-width', '1px')
    .attr('x1', x.bandwidth() / 2)
    .attr('x2', x.bandwidth() / 2)
    .attr('y1', -20)
    .attr('y2', -15);

  columnLabels.append('text')
    .attr('x', 10)
    .attr('y', y.bandwidth() / 2)
    .attr('dy', '.22em')
    .attr('text-anchor', 'end')
    .attr('transform', 'rotate(-90)')
    .text((d, i) => d);

  const rowLabels = labels.selectAll('.row-label')
    .data(labelsData)
    .enter().append('g')
      .attr('class', (d, i) => 'row-label row-' + i)
      .attr('transform', (d, i) => `translate(0, ${y(i.toString())})`);

  rowLabels.append('line')
    .style('stroke', 'black')
    .style('stroke-width', '1px')
    .attr('x1', 0)
    .attr('x2', -5)
    .attr('y1', y.bandwidth() / 2)
    .attr('y2', y.bandwidth() / 2);

  rowLabels.append('text')
    .attr('x', -11)
    .attr('y', y.bandwidth() / 2)
    .attr('dy', '.32em')
    .attr('text-anchor', 'end')
    .text((d, i) => d);

  // Y Axis label
  svg.append('text')
    .style('text-anchor', 'end')
    .attr('transform', 'translate(-5, -20)')
    .attr('class', 'strong')
    .text(options.yAxisLabel);

  // X Axis label
  svg.append('text')
    .attr('transform', `translate(${width}, ${height + margin.top - 10})`)
    .style('text-anchor', 'start')
    .attr('class', 'strong')
    .text(options.xAxisLabel);

  // Legend
  const legend = svg
    .append('defs')
    .append('svg:linearGradient')
    .attr('id', 'gradient')
    .attr('x1', '100%')
    .attr('y1', '0%')
    .attr('x2', '100%')
    .attr('y2', '100%')
    .attr('spreadMethod', 'pad');

  legend
    .append('stop')
    .attr('offset', '0%')
    .attr('stop-color', endColor)
    .attr('stop-opacity', 1);

  legend
    .append('stop')
    .attr('offset', '100%')
    .attr('stop-color', startColor)
    .attr('stop-opacity', 1);

  svg.append('rect')
    .attr('width', widthLegend / 2 - 10)
    .attr('height', height)
    .style('fill', 'url(#gradient)')
    .attr('transform', `translate(${width + 20}, 0)`);

  const legendY = d3.scaleLinear()
    .range([height, 0])
    .domain([minValue, maxValue]);

  svg.append('g')
    .attr('class', 'y axis')
    .attr('transform', `translate(${width + 60}, 0)`)
    .call(d3.axisRight(legendY) as any);
}

export function cellMouse(container: HTMLElement, cell: any, mouseIn: boolean) {
  const row = Number.parseInt(cell.parentElement.getAttribute('row'));
  const col = Number.parseInt(cell.getAttribute('column'));
  const rowLabel = container.querySelector(`.row-label.row-${row}`) as HTMLElement;
  const colLabel = container.querySelector(`.column-label.column-${col}`) as HTMLElement;

  // Hightlight row and column labels
  rowLabel.classList.toggle('active', mouseIn);
  colLabel.classList.toggle('active', mouseIn);

  // Highlight row and column cells
  const rowElement = container.querySelector(`.row[row='${row}']`) as HTMLElement;
  const colCells = container.querySelectorAll(`.cell[column='${col}']`) as NodeListOf<HTMLElement>;
  rowElement.classList.toggle('active', mouseIn);
  colCells.forEach((c) => c.classList.toggle('active', mouseIn));
}
