import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { NodePhase } from '../../model/argo_template';
import {
  FILTER_CHANGED_EVENT,
  FilterChangedEvent,
  ItemClickEvent,
  RouteEvent
} from '../../model/events';
import { JobMetadata } from '../../model/job';
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
    itemList.addEventListener(FILTER_CHANGED_EVENT, this._filterChanged.bind(this));
    itemList.addEventListener('itemDoubleClick', this._navigate.bind(this));
  }

  // TODO: should these jobs be cached?
  public async loadJobs(pipelineId: number, filterString?: string): Promise<void> {
    this._pipelineId = pipelineId;
    this.jobsMetadata = await Apis.getJobs(this._pipelineId, filterString);
    this.jobListRows = this.jobsMetadata.map((jobMetadata) => {
      const row = new ItemListRow({
        columns: [
          jobMetadata.name,
          new Date(jobMetadata.createdAt),
          new Date(jobMetadata.scheduledAt),
        ],
        selected: false,
      });
      return row;
    });
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

  private _filterChanged(ev: FilterChangedEvent): void {
    // This function will wait 300ms after last time it is called (last
    // keystroke in filter box) before getJobs() is called.
    this._keystrokeDebouncer = Polymer.Debouncer.debounce(
        this._keystrokeDebouncer,
        Polymer.Async.timeOut.after(300),
        async () => { this.loadJobs(this._pipelineId, ev.detail.filterString); }
    );
    // Allows tests to use Polymer.flush to ensure debounce has completed.
    Polymer.enqueueDebouncer(this._keystrokeDebouncer);
  }
}
