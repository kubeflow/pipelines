import * as d3 from 'd3';

export interface MatrixOptions {
  data: number[][];
  container: HTMLElement;
  labels: string[];
  startColor: string;
  endColor: string;
}

const margin = { top: 50, right: 50, bottom: 100, left: 100 };

/**
 * Draws a confusion matrix with the given options. One of the options
 * is the HTMLElement container, which gets populated with the matrix
 * elements.
 */
export function drawMatrix(options: MatrixOptions) {
  const width = 250;
  const height = 250;
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
    .attr('width', width + margin.left + margin.right)
    .attr('height', height + margin.top + margin.bottom)
    .append('g')
    .attr('transform', 'translate(' + margin.left + ',' + margin.top + ')');

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
    .attr('transform', (d, i) => 'translate(0,' + y(i.toString()) + ')');

  const cell = row.selectAll('.cell')
    .data((d) => d)
    .enter().append('g')
    .attr('class', 'cell')
    .attr('transform', (d, i) => 'translate(' + x(i.toString()) + ', 0)');

  cell.append('rect')
    .attr('width', x.bandwidth())
    .attr('height', y.bandwidth())
    .style('stroke-width', 0);

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
    .attr('class', 'column-label')
    .attr('transform', (d, i) => 'translate(' + x(i.toString()) + ',' + height + ')');

  columnLabels.append('line')
    .style('stroke', 'black')
    .style('stroke-width', '1px')
    .attr('x1', x.bandwidth() / 2)
    .attr('x2', x.bandwidth() / 2)
    .attr('y1', 0)
    .attr('y2', 5);

  columnLabels.append('text')
    .attr('x', 20)
    .attr('y', y.bandwidth() / 2)
    .attr('dy', '.22em')
    .attr('text-anchor', 'end')
    .attr('transform', 'rotate(-60)')
    .text((d, i) => d);

  const rowLabels = labels.selectAll('.row-label')
    .data(labelsData)
    .enter().append('g')
    .attr('class', 'row-label')
    .attr('transform', (d, i) => 'translate(' + 0 + ',' + y(i.toString()) + ')');

  rowLabels.append('line')
    .style('stroke', 'black')
    .style('stroke-width', '1px')
    .attr('x1', 0)
    .attr('x2', -5)
    .attr('y1', y.bandwidth() / 2)
    .attr('y2', y.bandwidth() / 2);

  rowLabels.append('text')
    .attr('x', -8)
    .attr('y', y.bandwidth() / 2)
    .attr('dy', '.32em')
    .attr('text-anchor', 'end')
    .text((d, i) => d);

  const key = d3.select(container)
    .append('svg')
    .attr('width', widthLegend)
    .attr('height', height + margin.top + margin.bottom);

  const legend = key
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

  key.append('rect')
    .attr('width', widthLegend / 2 - 10)
    .attr('height', height)
    .style('fill', 'url(#gradient)')
    .attr('transform', 'translate(0,' + margin.top + ')');

  d3.scaleLinear()
    .range([height, 0])
    .domain([minValue, maxValue]);

  const legendY = d3.scaleLinear()
    .range([height, 0])
    .domain([minValue, maxValue]);

  key.append('g')
    .attr('class', 'y axis')
    .attr('transform', 'translate(41,' + margin.top + ')')
    .call(d3.axisRight(legendY) as any);
}

/**
 * Draws a statistics table with prevision, recall, and accuracy.
 */
export function drawStatsTable(container: HTMLElement, data: number[][]) {
  if (!Array.isArray(data) || !data.length || !Array.isArray(data[0])) {
    throw new Error('Confusion matrix input should be a 2D array.');
  }

  const columns = ['Precision', 'Recall', 'Accuracy'];

  const tp = data[0][0];
  const fn = data[0][1];
  const fp = data[1][0];
  const tn = data[1][1];

  const p = tp + fn;
  const n = fp + tn;

  let accuracy = (tp + tn) / (p + n);
  let precision = tp / (tp + fp);
  let recall = tp / (tp + fn);

  accuracy = Math.round(accuracy * 100) / 100;
  precision = Math.round(precision * 100) / 100;
  recall = Math.round(recall * 100) / 100;

  const tableData = [];
  tableData.push({
    Accuracy: accuracy,
    Precision: precision,
    Recall: recall,
  });

  const table = d3.select(container).append('table')
    .attr('style', 'margin-left: ' + margin.left + 'px');
  const thead = table.append('thead');
  const tbody = table.append('tbody');

  // append the header row
  thead.append('tr')
    .selectAll('th')
    .data(columns)
    .enter()
    .append('th')
    .text((column) => column);

  // create a row for each object in the data
  const rows = tbody.selectAll('tr')
    .data(tableData)
    .enter()
    .append('tr');

  // create a cell in each row for each column
  rows.selectAll('td')
    .data((row) => {
      return columns.map((column) => {
        return { column, value: (row as any)[column] };
      });
    })
    .enter()
    .append('td')
    .attr('style', 'font-family: monospace') // sets the font style
    .html((d) => d.value);

  return table;
}
