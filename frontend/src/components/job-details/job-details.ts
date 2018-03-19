import 'iron-icons/iron-icons.html';
import 'paper-progress/paper-progress.html';
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';
import '../data-plotter/data-plot';

import * as jsYaml from 'js-yaml';
import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

// @ts-ignore
import prettyJson from 'json-pretty-html';
import { customElement, property } from '../../decorators';
import { PageElement } from '../../lib/page_element';
import { parseTemplateOuputPaths } from '../../lib/template_parser';
import { NodePhase } from '../../model/argo_template';
import { Job } from '../../model/job';
import { PlotMetadata } from '../../model/output_metadata';
import { JobGraph } from '../job-graph/job-graph';

import '../job-graph/job-graph';
import './job-details.html';

@customElement('job-details')
export class JobDetails extends Polymer.Element implements PageElement {

  @property({ type: Array })
  public outputPlots: PlotMetadata[] = [];

  @property({ type: Object })
  public job: Job | null = null;

  @property({ type: Number })
  public selectedTab = 0;

  @property({ type: Number })
  protected _pipelineId = -1;

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

      // Get the job graph
      const runtimeGraphYaml = await Apis.getJobRuntimeTemplate(this._pipelineId, this._jobId);
      const runtimeGraph = jsYaml.safeLoad(runtimeGraphYaml);
      (this.$.jobGraph as JobGraph).refresh(runtimeGraph);
    }
  }

  protected _dateToString(date: number) {
    return Utils.dateToString(date);
  }

  protected _getStatusIcon(status: NodePhase) {
    return Utils.nodePhaseToIcon(status);
  }

  protected _getRuntime(start: number, end: number, status: NodePhase) {
    if (!status) {
      return '-';
    }
    if (end === -1) {
      end = Date.now();
    }
    return Utils.dateDiffToString(end - start);
  }

  protected _getProgressColor(status: NodePhase) {
    return Utils.nodePhaseToColor(status);
  }
}
