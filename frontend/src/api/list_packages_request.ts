export class ListPackagesRequest {
  public orderAscending: boolean;
  public pageSize?: number;
  public pageToken: string;
  public sortBy: string;

  constructor(pageSize?: number) {
    this.pageSize = pageSize;
    this.pageToken = '';
    this.sortBy = '';
    this.orderAscending = true;
  }

  public toQueryParams(): string {
    const params = [
      `ascending=${this.orderAscending}`,
      `pageToken=${this.pageToken}`,
      `sortBy=${encodeURIComponent(this.sortBy)}`,
    ];
    if (this.pageSize !== undefined) {
      params.push(`pageSize=${this.pageSize}`);
    }
    return params.join('&');
  }
}
