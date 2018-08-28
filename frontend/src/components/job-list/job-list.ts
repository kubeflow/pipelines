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

import 'iron-icons/iron-icons.html';
import 'paper-button/paper-button.html';
import 'paper-spinner/paper-spinner.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import * as xss from 'xss';
import { Job } from '../../api/job';
import { JobSortKeys, ListJobsRequest } from '../../api/list_jobs_request';
import { ListRunsRequest } from '../../api/list_runs_request';
import { DialogResult } from '../../components/popup-dialog/popup-dialog';
import {
  ItemDblClickEvent,
  ListFormatChangeEvent,
  NewListPageEvent,
  RouteEvent,
} from '../../model/events';
import { PageElement } from '../../model/page_element';
import {
  ColumnType,
  ColumnTypeName,
  ItemListColumn,
  ItemListElement,
  ItemListRow,
} from '../item-list/item-list';

import './job-list.html';

@customElement('job-list')
export class JobList extends PageElement {

  @property({ type: Array })
  public jobs: Job[] = [];

  @property({ type: Boolean })
  protected _busy = false;

  @property({ type: Boolean })
  protected _oneItemIsSelected = false;

  @property({ type: Boolean })
  protected _atLeastOneItemIsSelected = false;

  public get newButton(): PaperButtonElement {
    return this.$.newBtn as PaperButtonElement;
  }

  public get refreshButton(): PaperButtonElement {
    return this.$.refreshBtn as PaperButtonElement;
  }

  public get cloneButton(): PaperButtonElement {
    return this.$.cloneBtn as PaperButtonElement;
  }

  public get deleteButton(): PaperButtonElement {
    return this.$.deleteBtn as PaperButtonElement;
  }

  public get itemList(): ItemListElement {
    return this.$.jobsItemList as ItemListElement;
  }

  protected jobListRows: ItemListRow[] = [];

  protected jobListColumns: ItemListColumn[] = [
    new ItemListColumn('Name', ColumnTypeName.STRING, JobSortKeys.NAME, 2),
    new ItemListColumn('Last 5 runs', ColumnTypeName.STRING, undefined, 1),
    // Sorting by Pipeline is not supported right now because we are displaying Pipeline name, but
    // the backend giving us Pipeline IDs. See: (#903) and (#904)
    new ItemListColumn('Pipeline', ColumnTypeName.STRING, undefined, 1.5),
    new ItemListColumn('Created at', ColumnTypeName.DATE, JobSortKeys.CREATED_AT),
    new ItemListColumn('Schedule', ColumnTypeName.STRING),
    new ItemListColumn('Enabled', ColumnTypeName.STRING, undefined, 0.5),
  ];

  private _debouncer: Polymer.Debouncer | undefined = undefined;

  public ready(): void {
    super.ready();
    this.itemList.addEventListener(ListFormatChangeEvent.name, this._listFormatChanged.bind(this));
    this.itemList.addEventListener(NewListPageEvent.name, this._loadNewListPage.bind(this));
    this.itemList.addEventListener('selected-indices-changed',
        this._selectedItemsChanged.bind(this));
    this.itemList.addEventListener(ItemDblClickEvent.name, this._navigate.bind(this));

    this.itemList.renderColumn = (value: ColumnType, colIndex: number, rowIndex: number) => {
      if (value === undefined) {
        return '-';
      }
      let text = xss(value.toString());
      switch (colIndex) {
        case 0:
          return `<a class="link" href="/jobs/details/${this.jobs[rowIndex].id}">
                    ${text}
                  </a>`;
        case 1:
          const statuses = text.split(':').filter((s) => !!s);
          while (statuses.length < 5) {
            statuses.push('NONE');
          }
          return statuses.map((status) =>
              `<iron-icon icon=${Utils.nodePhaseToIcon(status as any)} title="${status}"
                class="padded-spacing-minus-6"></iron-icon>`).join('');
        default:
          if (this.itemList.columns[colIndex] && value &&
              this.itemList.columns[colIndex].type === ColumnTypeName.DATE) {
            text = (value as Date).toLocaleString();
          }
          return text;
      }
    };
  }

  public load(): void {
    this.itemList.reset();
    this._loadJobs(new ListJobsRequest(this.itemList.selectedPageSize));
  }

  protected _navigate(ev: ItemDblClickEvent): void {
    const jobId = this.jobs[ev.detail.index].id;
    this.dispatchEvent(new RouteEvent(`/jobs/details/${jobId}`));
  }

  protected _refresh(): void {
    this.load();
  }

  protected _cloneJob(): void {
    // Clone Job button is only enabled if there is one selected item.
    const selectedJob = this.jobs[this.itemList.selectedIndices[0]];
    this.dispatchEvent(
        new RouteEvent(
          '/jobs/new',
          {
            parameters: selectedJob.parameters,
            pipelineId: selectedJob.pipeline_id,
          }));
  }

  protected async _deleteJob(): Promise<void> {
    const deletedItemsLen = this.itemList.selectedIndices.length;
    const pluralS = deletedItemsLen > 1 ? 's' : '';
    const dialogResult = await Utils.showDialog(
        `Delete ${deletedItemsLen} job${pluralS}?`,
        `You are about to delete ${deletedItemsLen} job${pluralS}.
         Are you sure you want to proceed?`,
        `Delete ${deletedItemsLen} job${pluralS}`,
        'Cancel');

    // BUTTON1 is Delete
    if (dialogResult !== DialogResult.BUTTON1) {
      return;
    }

    this._busy = true;
    let unsuccessfulDeletes = 0;
    let errorMessage = '';

    await Promise.all(this.itemList.selectedIndices.map(async (i) => {
      try {
        await Apis.deleteJob(this.jobs[i].id);
      } catch (err) {
        errorMessage = `Deleting Job: "${this.jobs[i].name}" failed with error: "${err}"`;
        unsuccessfulDeletes++;
      }
    }));

    const successfulDeletes = this.itemList.selectedIndices.length - unsuccessfulDeletes;
    if (successfulDeletes > 0) {
      Utils.showNotification(`Successfully deleted ${successfulDeletes} Jobs!`);
      this.itemList.reset();
      this._loadJobs(new ListJobsRequest(this.itemList.selectedPageSize));
    }

    if (unsuccessfulDeletes > 0) {
      Utils.showDialog(`Failed to delete ${unsuccessfulDeletes} Jobs`, errorMessage, 'Dismiss');
    }

    this._busy = false;
  }

  protected _newJob(): void {
    this.dispatchEvent(new RouteEvent('/jobs/new'));
  }

  private _loadNewListPage(ev: NewListPageEvent): void {
    const request = new ListJobsRequest(ev.detail.pageSize);
    request.filterBy = ev.detail.filterBy;
    request.pageToken = ev.detail.pageToken;
    request.sortBy = ev.detail.sortBy;

    this._loadJobs(request);
  }

  private _selectedItemsChanged(): void {
    if (this.itemList.selectedIndices) {
      this._oneItemIsSelected = this.itemList.selectedIndices.length === 1;
      this._atLeastOneItemIsSelected = this.itemList.selectedIndices.length > 0;
    } else {
      this._oneItemIsSelected = false;
      this._atLeastOneItemIsSelected = false;
    }
  }

  private _listFormatChanged(ev: ListFormatChangeEvent): void {
    // This function will wait 300ms after last time it is called before listJobs() is called.
    this._debouncer = Polymer.Debouncer.debounce(
        this._debouncer || null,
        Polymer.Async.timeOut.after(300),
        async () => {
          const request = new ListJobsRequest(ev.detail.pageSize);
          request.filterBy = ev.detail.filterString;
          request.orderAscending = ev.detail.orderAscending;
          request.sortBy = ev.detail.sortColumn;
          this._loadJobs(request);
        }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._debouncer);
  }

  private async _loadJobs(request: ListJobsRequest): Promise<void> {
    try {
      const listJobsResponse = await Apis.listJobs(request);
      this.jobs = listJobsResponse.jobs || [];

      this.itemList.updateNextPageToken(listJobsResponse.next_page_token || '');
    } catch (err) {
      this.showPageError('There was an error while loading the job list', err.message);
      Utils.log.verbose('Error loading jobs:', err);
    }

    this.jobListRows = this.jobs.map((job) => {
      // TODO: we should just call job.trigger.toString() here, but the lack of types in
      // the mocked data prevents us from being able to use functions at the moment.
      let schedule = '-';
      if (job && job.trigger) {
        schedule = job.trigger.toString();
      }
      const row = new ItemListRow({
        columns: [
          job.name,
          '',
          '',
          Utils.formatDateString(job.created_at),
          schedule,
          Utils.enabledDisplayString(job.trigger, job.enabled)
        ],
        selected: false,
      });
      return row;
    });

    // Fetch and set last 5 runs' statuses for each job
    this.jobs.forEach(async (job, i) => {
      const listRunsResponse = await Apis.listRuns(new ListRunsRequest(job.id, 5));
      this.set(`jobListRows.${i}.columns.1`,
          (listRunsResponse.runs || []).map((r) => r.status).join(':'));
    });

    // Fetch and set pipeline name for each job
    this.jobs.forEach(async (job, i) => {
      const pipeline = await Apis.getPipeline(job.pipeline_id);
      this.set(`jobListRows.${i}.columns.2`, pipeline.name);
    });
  }
}
