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
import 'paper-tabs/paper-tab.html';
import 'paper-tabs/paper-tabs.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, observe, property } from 'polymer-decorators/src/decorators';
import * as xss from 'xss';
import { apiJob } from '../../api/job';
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
import '../run-list/run-list';
import { RunList } from '../run-list/run-list';

import './job-list.html';

@customElement('job-list')
export class JobList extends PageElement {

  @property({ type: Array })
  public jobs: apiJob[] = [];

  @property({ type: Number })
  public selectedTab = -1;

  @property({ type: Boolean })
  protected _busy = false;

  @property({ type: String })
  protected _runsTabHref = '#allRuns';

  @property({ type: String })
  protected _jobsTabHref = '#groupByJob';

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

  public get jobsItemList(): ItemListElement {
    return this.$.jobsItemList as ItemListElement;
  }

  public get runsItemList(): RunList {
    return this.$.runsItemList as RunList;
  }

  public get tabs(): PaperTabsElement {
    return this.$.tabs as PaperTabsElement;
  }

  protected jobListRows: ItemListRow[] = [];

  protected jobListColumns: ItemListColumn[] = [
    new ItemListColumn('Name', ColumnTypeName.STRING, Apis.JobSortKeys.NAME, 2),
    new ItemListColumn('Last 5 runs', ColumnTypeName.STRING, undefined, 1),
    // Sorting by Pipeline is not supported right now because we are displaying Pipeline name, but
    // the backend giving us Pipeline IDs. See: (#903) and (#904)
    new ItemListColumn('Pipeline', ColumnTypeName.STRING, undefined, 1.5),
    new ItemListColumn('Created at', ColumnTypeName.DATE, Apis.JobSortKeys.CREATED_AT),
    new ItemListColumn('Schedule', ColumnTypeName.STRING),
    new ItemListColumn('Enabled', ColumnTypeName.STRING, undefined, 0.5),
  ];

  private _debouncer: Polymer.Debouncer | undefined = undefined;

  public ready(): void {
    super.ready();
    this.jobsItemList.addEventListener(
        ListFormatChangeEvent.name, this._listFormatChanged.bind(this));
    this.jobsItemList.addEventListener(NewListPageEvent.name, this._loadNewListPage.bind(this));
    this.jobsItemList.addEventListener('selected-indices-changed',
        this._selectedItemsChanged.bind(this));
    this.jobsItemList.addEventListener(ItemDblClickEvent.name, this._navigate.bind(this));

    this.jobsItemList.renderColumn = (value: ColumnType, colIndex: number, rowIndex: number) => {
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
          const statuses = text.split(',').filter((s) => !!s);
          while (statuses.length < 5) {
            statuses.push('NONE');
          }
          return statuses.map((status) =>
              `<iron-icon icon=${Utils.nodePhaseToIcon(status as any)} title="${status}"
                class="padded-spacing-minus-6"></iron-icon>`).join('');
        default:
          if (this.jobsItemList.columns[colIndex] && value &&
              this.jobsItemList.columns[colIndex].type === ColumnTypeName.DATE) {
            text = (value as Date).toLocaleString();
          }
          return text;
      }
    };
  }

  public load(): void {
    this.jobsItemList.reset();
    const tabElement = this.tabs.querySelector(`[href="${location.hash}"]`);
    this.selectedTab = tabElement ? this.tabs.indexOf(tabElement) : 0;
    this._loadSelectedPage();
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
    const selectedJob = this.jobs[this.jobsItemList.selectedIndices[0]];
    this.dispatchEvent(
        new RouteEvent(
          '/jobs/new',
          {
            parameters: selectedJob.parameters,
            pipelineId: selectedJob.pipeline_id,
          }));
  }

  protected async _deleteJob(): Promise<void> {
    const deletedItemsLen = this.jobsItemList.selectedIndices.length;
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

    await Promise.all(this.jobsItemList.selectedIndices.map(async (i) => {
      try {
        await Apis.deleteJob(this.jobs[i].id!);
      } catch (err) {
        errorMessage = `Deleting Job: "${this.jobs[i].name}" failed with error: "${err}"`;
        unsuccessfulDeletes++;
      }
    }));

    const successfulDeletes = this.jobsItemList.selectedIndices.length - unsuccessfulDeletes;
    if (successfulDeletes > 0) {
      Utils.showNotification(`Successfully deleted ${successfulDeletes} Jobs!`);
      this.jobsItemList.reset();
      this._loadJobs({ pageSize: this.jobsItemList.selectedPageSize });
    }

    if (unsuccessfulDeletes > 0) {
      Utils.showDialog(`Failed to delete ${unsuccessfulDeletes} Jobs`, errorMessage, 'Dismiss');
    }

    this._busy = false;
  }

  protected _newJob(): void {
    this.dispatchEvent(new RouteEvent('/jobs/new'));
  }

  @observe('selectedTab')
  protected _selectedTabChanged(): void {
    if (this.selectedTab === -1) {
      return;
    }
    const tab = (this.tabs.selectedItem || this.tabs.children[0]) as PaperTabElement | undefined;
    if (tab) {
      const href = tab.getAttribute('href');
      location.hash = href || '';
      this._loadSelectedPage();
    }
  }

  private _loadSelectedPage(): void {
    if (this.selectedTab === 1) {
      this.runsItemList.loadRuns();
    } else {
      this._loadJobs({ pageSize: this.jobsItemList.selectedPageSize });
    }
  }

  private _loadNewListPage(ev: NewListPageEvent): void {
    this._loadJobs({
      filterBy: ev.detail.filterBy,
      pageSize: ev.detail.pageSize,
      pageToken: ev.detail.pageToken,
      sortBy: ev.detail.sortBy,
    });
  }

  private _selectedItemsChanged(): void {
    if (this.jobsItemList.selectedIndices) {
      this._oneItemIsSelected = this.jobsItemList.selectedIndices.length === 1;
      this._atLeastOneItemIsSelected = this.jobsItemList.selectedIndices.length > 0;
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
          this._loadJobs({
            filterBy: ev.detail.filterString,
            orderAscending: ev.detail.orderAscending,
            pageSize: ev.detail.pageSize,
            sortBy: ev.detail.sortColumn,
          });
        }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._debouncer);
  }

  private async _loadJobs(request: Apis.ListJobsRequest): Promise<void> {
    try {
      const listJobsResponse = await Apis.listJobs(request);
      this.jobs = listJobsResponse.jobs || [];

      this.jobsItemList.updateNextPageToken(listJobsResponse.next_page_token || '');
    } catch (err) {
      this.showPageError('There was an error while loading the job list', err.message);
      Utils.log.verbose('Error loading jobs:', err);
    }

    this.jobListRows = this.jobs.map((job) => {
      // TODO: we should just call job.trigger.toString() here, but the lack of types in
      // the mocked data prevents us from being able to use functions at the moment.
      let schedule = '-';
      if (job && job.trigger) {
        schedule = Utils.triggerDisplayString(job.trigger);
      }
      const row = new ItemListRow({
        columns: [
          job.name,
          '',
          '',
          Utils.formatDateString(job.created_at),
          schedule,
          Utils.enabledDisplayString(job.trigger, job.enabled || false)
        ],
        selected: false,
      });
      return row;
    });

    // Fetch and set last 5 runs' statuses for each job
    this.jobs.forEach(async (job, i) => {
      const listRunsResponse = await Apis.listRuns({ jobId: job.id, pageSize: 5 });
      const statusList = (listRunsResponse.runs || []).map((r) =>
          Utils.getLastInStatusList(r.status || ''));
      this.set(`jobListRows.${i}.columns.1`, statusList.join(','));
    });

    // Fetch and set pipeline name for each job
    this.jobs.forEach(async (job, i) => {
      const pipeline = await Apis.getPipeline(job.pipeline_id!);
      this.set(`jobListRows.${i}.columns.2`, pipeline.name);
    });
  }
}
