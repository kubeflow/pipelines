import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { NodePhase } from '../../model/argo_template';
import {
  EventName,
  FilterChangedEvent,
  ItemClickEvent,
  NewListPageEvent,
  RouteEvent
} from '../../model/events';
import { JobMetadata } from '../../model/job';
import { ListJobsRequest } from '../../model/list_jobs_request';
import {
  ColumnTypeName,
  ItemListColumn,
  ItemListElement,
  ItemListRow
} from '../item-list/item-list';

import './job-list.html';

@customElement('job-list')
export class JobList extends Polymer.Element {

  @property({ type: Array })
  public jobsMetadata: JobMetadata[] = [];

  @property({ type: Number })
  protected _pageSize = 20;

  protected jobListRows: ItemListRow[] = [];

  protected jobListColumns: ItemListColumn[] = [
    { name: 'Job Name', type: ColumnTypeName.STRING },
    { name: 'Created at', type: ColumnTypeName.DATE },
    { name: 'Scheduled at', type: ColumnTypeName.DATE },
  ];

  private _keystrokeDebouncer: Polymer.Debouncer;

  private _pipelineId = -1;

  ready(): void {
    super.ready();
    const itemList = this.$.jobsItemList as ItemListElement;
    itemList.addEventListener(EventName.FILTER_CHANGED, this._filterChanged.bind(this));
    itemList.addEventListener(EventName.NEW_LIST_PAGE, this._loadNewListPage.bind(this));
    itemList.addEventListener('itemDoubleClick', this._navigate.bind(this));
  }

  public loadJobs(pipelineId: number): void {
    this._pipelineId = pipelineId;
    (this.$.jobsItemList as ItemListElement).reset();
    this._loadJobsInternal(new ListJobsRequest(pipelineId, this._pageSize));
  }

  protected _navigate(ev: ItemClickEvent): void {
    const jobId = this.jobsMetadata[ev.detail.index].name;
    this.dispatchEvent(
        new RouteEvent(`/pipelineJob?pipelineId=${this._pipelineId}&jobId=${jobId}`));
  }

  protected _getStatusIcon(status: NodePhase): string {
    return Utils.nodePhaseToIcon(status);
  }

  protected _getRuntime(start: string, end: string, status: NodePhase): string {
    if (!status) {
      return '-';
    }
    const startDate = new Date(start);
    const endDate = end ? new Date(end) : new Date();
    return Utils.dateDiffToString(endDate.valueOf() - startDate.valueOf());
  }

  private async _loadJobsInternal(request: ListJobsRequest): Promise<void> {
    try {
      const getJobsResponse = await Apis.getJobs(request);
      this.jobsMetadata = getJobsResponse.jobs;

      const itemList = this.$.jobsItemList as ItemListElement;
      itemList.updateNextPageToken(getJobsResponse.nextPageToken);
    } catch (err) {
      // TODO: This error should be bubbled up to pipeline-details to be shown as a page error.
      Utils.showDialog('There was an error while loading the job list', err);
    }

    this.jobListRows = this.jobsMetadata.map((jobMetadata) => {
      const row = new ItemListRow({
        columns: [
          jobMetadata.name,
          Utils.formatDateInSeconds(jobMetadata.createdAt),
          Utils.formatDateInSeconds(jobMetadata.scheduledAt),
        ],
        selected: false,
      });
      return row;
    });
  }

  private _loadNewListPage(ev: NewListPageEvent): void {
    const request = new ListJobsRequest(this._pipelineId, this._pageSize);
    request.filterBy = ev.detail.filterBy;
    request.pageToken = ev.detail.pageToken;
    request.sortBy = ev.detail.sortBy;
    this._loadJobsInternal(request);
  }

  private _filterChanged(ev: FilterChangedEvent): void {
    // This function will wait 300ms after last time it is called (last
    // keystroke in filter box) before getJobs() is called.
    this._keystrokeDebouncer = Polymer.Debouncer.debounce(
        this._keystrokeDebouncer,
        Polymer.Async.timeOut.after(300),
        async () => {
          const request = new ListJobsRequest(this._pipelineId, this._pageSize);
          request.filterBy = ev.detail.filterString;
          this._loadJobsInternal(request);
        }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._keystrokeDebouncer);
  }
}
