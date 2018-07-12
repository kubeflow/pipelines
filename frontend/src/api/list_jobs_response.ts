import { JobMetadata } from './job';

export class ListJobsResponse {
  public jobs?: JobMetadata[];
  public next_page_token?: string;

  public static buildFromObject(listJobsResponse: any): ListJobsResponse {
    const newListJobsResponse = new ListJobsResponse();
    newListJobsResponse.next_page_token = listJobsResponse.next_page_token;
    if (listJobsResponse.jobs && listJobsResponse.jobs.length > 0) {
      newListJobsResponse.jobs =
          listJobsResponse.jobs.map((j: any) => JobMetadata.buildFromObject(j));
    }
    return newListJobsResponse;
  }
}
