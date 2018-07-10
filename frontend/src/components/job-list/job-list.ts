import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { JobMetadata } from '../../api/job';
import { JobSortKeys, ListJobsRequest } from '../../api/list_jobs_request';
import { NodePhase } from '../../model/argo_template';
import {
  ItemDblClickEvent,
  ListFormatChangeEvent,
  NewListPageEvent,
  RouteEvent,
} from '../../model/events';
import {
  ColumnTypeName,
  ItemListColumn,
  ItemListElement,
  ItemListRow,
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
    new ItemListColumn('Job Name', ColumnTypeName.STRING, JobSortKeys.NAME),
    new ItemListColumn('Created at', ColumnTypeName.DATE, JobSortKeys.CREATED_AT),
    new ItemListColumn('Scheduled at', ColumnTypeName.DATE),
  ];

  private _debouncer: Polymer.Debouncer;

  private _pipelineId = -1;

  public ready(): void {
    super.ready();
    const itemList = this.$.jobsItemList as ItemListElement;
    itemList.addEventListener(ListFormatChangeEvent.name, this._listFormatChanged.bind(this));
    itemList.addEventListener(NewListPageEvent.name, this._loadNewListPage.bind(this));
    itemList.addEventListener(ItemDblClickEvent.name, this._navigate.bind(this));
  }

  public loadJobs(pipelineId: number): void {
    this._pipelineId = pipelineId;
    (this.$.jobsItemList as ItemListElement).reset();
    this._loadJobsInternal(new ListJobsRequest(pipelineId, this._pageSize));
  }

  protected _navigate(ev: ItemDblClickEvent): void {
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
      this.jobsMetadata = getJobsResponse.jobs || [];

      const itemList = this.$.jobsItemList as ItemListElement;
      itemList.updateNextPageToken(getJobsResponse.nextPageToken || '');
    } catch (err) {
      // TODO: This error should be bubbled up to pipeline-details to be shown as a page error.
      Utils.showDialog('There was an error while loading the job list', err);
    }

    this.jobListRows = this.jobsMetadata.map((jobMetadata) => {
      const row = new ItemListRow({
        columns: [
          jobMetadata.name,
          Utils.formatDateString(jobMetadata.created_at),
          Utils.formatDateString(jobMetadata.scheduled_at),
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

  private _listFormatChanged(ev: ListFormatChangeEvent): void {
    // This function will wait 300ms after last time it is called before getJobs() is called.
    this._debouncer = Polymer.Debouncer.debounce(
        this._debouncer,
        Polymer.Async.timeOut.after(300),
        async () => {
          const request = new ListJobsRequest(this._pipelineId, this._pageSize);
          request.filterBy = ev.detail.filterString;
          request.orderAscending = ev.detail.orderAscending;
          request.sortBy = ev.detail.sortColumn;
          this._loadJobsInternal(request);
        }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._debouncer);
  }
}
