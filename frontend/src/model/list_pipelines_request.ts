export class ListPipelinesRequest {
  public filterBy: string;
  public pageSize: number;
  public pageToken: string;
  public sortBy: string;

  constructor(pageSize: number) {
    this.filterBy = '';
    this.pageSize = pageSize;
    this.pageToken = '';
    this.sortBy = '';
  }
}
