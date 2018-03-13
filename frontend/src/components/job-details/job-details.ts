import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';
import '../data-plotter/data-plot';

import * as dagre from 'dagre';
import * as jsYaml from 'js-yaml';
import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

// @ts-ignore
import prettyJson from 'json-pretty-html';
import { customElement, property } from '../../decorators';
import { PageElement } from '../../lib/page_element';
import { parseTemplateOuputPaths } from '../../lib/template_parser';
import { Job, JobStatus } from '../../model/job';
import { PlotMetadata } from '../../model/output_metadata';

import './job-details.html';

const progressCssColors = {
  completed: '--success-color',
  errored: '--error-color',
  notStarted: '',
  running: '--progress-color',
};

interface Line {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  distance: number;
  xMid: number;
  yMid: number;
  angle: number;
  left: number;
}

interface Edge {
  from: string;
  to: string;
  lines: Line[];
}

@customElement('job-details')
export class JobDetails extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public outputPlots: PlotMetadata[] = [];

  @property({ type: Object })
  public job: Job | null = null;

  @property({ type: Number })
  public selectedTab = 0;

  @property({ type: Array })
  protected _workflowNodes: dagre.Node[] = [];

  @property({ type: Array })
  protected _workflowEdges: Edge[] = [];

  private _pipelineId = -1;
  private _jobId = '';

  public async refresh(_: string, queryParams: { jobId?: string, pipelineId: number }) {
    if (queryParams.jobId !== undefined && queryParams.pipelineId > -1) {
      this._pipelineId = queryParams.pipelineId;
      this._jobId = queryParams.jobId;
      this.job = await Apis.getJob(this._pipelineId, this._jobId);

      const pipeline = await Apis.getPipeline(this._pipelineId);
      const templateYaml = await Apis.getPackageTemplate(pipeline.packageId);

      const baseOutputPath = pipeline
        .parameters
        .filter((p) => p.name === 'output')[0]
        .value.toString();

      const outputPaths = parseTemplateOuputPaths(templateYaml, baseOutputPath, this._jobId);

      // Clear outputPlots to keep from re-adding the same outputs over and over.
      this.set('outputPlots', []);

      outputPaths.forEach(async (path) => {
        const fileList = await Apis.listFiles(path);
        const metadataFile = fileList.filter((f) => f.endsWith('/metadata.json'))[0];
        if (metadataFile) {
          const metadataJson = await Apis.readFile(metadataFile);
          this.push('outputPlots', JSON.parse(metadataJson) as PlotMetadata);
        }
      });

      this._colorProgressBar();

      // Get the job graph
      const graphYaml = jsYaml.safeLoad(await Apis.getJobGraph(queryParams.jobId));
      console.log(graphYaml);

      const g = new dagre.graphlib.Graph();
      g.setGraph({});
      g.setDefaultEdgeLabel(() => ({}));
      g.setNode('kspacey', { label: 'Kevin Spacey', width: 182, height: 52 });
      g.setNode('swilliams', { label: 'Saul Williams', width: 182, height: 52 });
      g.setNode('bpitt', { label: 'Brad Pitt', width: 182, height: 52 });
      g.setNode('hford', { label: 'Harrison Ford', width: 182, height: 52 });
      g.setNode('lwilson', { label: 'Luke Wilson', width: 182, height: 52 });
      g.setNode('kbacon', { label: 'Kevin Bacon', width: 182, height: 52 });

      // Add edges to the graph.
      g.setEdge('kspacey', 'swilliams');
      g.setEdge('swilliams', 'kbacon');
      g.setEdge('bpitt', 'kbacon');
      g.setEdge('hford', 'lwilson');
      g.setEdge('lwilson', 'kbacon');

      dagre.layout(g);

      this._workflowNodes = g.nodes().map((id) => g.node(id));

      g.edges().forEach((edgeInfo) => {
        const edge = g.edge(edgeInfo);
        const lines: Line[] = [];
        if (edge.points.length > 1) {
          for (let i = 1; i < edge.points.length; i++) {
            const x1 = edge.points[i - 1].x;
            const y1 = edge.points[i - 1].y;
            const x2 = edge.points[i].x;
            const y2 = edge.points[i].y;
            const distance = Math.sqrt(Math.pow(x1 - x2, 2) + Math.pow(y1 - y2, 2));
            const xMid = (x1 + x2) / 2;
            const yMid = (y1 + y2) / 2;
            const angle = Math.atan2(y1 - y2, x1 - x2) * 180 / Math.PI;
            const left = xMid - (distance / 2);
            lines.push({ x1, y1, x2, y2, distance, xMid, yMid, angle, left });
          }
        }
        this.push('_workflowEdges', { from: edgeInfo.v, to: edgeInfo.w, lines });
      });
    }
  }

  protected _dateToString(date: number) {
    return Utils.dateToString(date);
  }

  protected _getStatusIcon(status: JobStatus) {
    return Utils.jobStatusToIcon(status);
  }

  protected _getRuntime(start: number, end: number, status: JobStatus) {
    if (status === 'Not started') {
      return '-';
    }
    if (end === -1) {
      end = Date.now();
    }
    return Utils.dateDiffToString(end - start);
  }

  private _colorProgressBar() {
    if (!this.job) {
      return;
    }
    let color = '';
    switch (this.job.status) {
      case 'Running':
        color = progressCssColors.running;
        break;
      case 'Succeeded':
        color = progressCssColors.completed;
        break;
      case 'Errored':
        color = progressCssColors.errored;
      default:
        color = progressCssColors.notStarted;
        break;
    }
    (this.$.progress as any).updateStyles({
      '--paper-progress-active-color': `var(${color})`,
    });
  }
}
