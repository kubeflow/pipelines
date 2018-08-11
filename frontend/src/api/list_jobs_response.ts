import { Job } from './job';

export class ListJobsResponse {
  public next_page_token?: string;
  public jobs?: Job[];

  public static buildFromObject(listJobsResponse: any): ListJobsResponse {
    const newListJobsResponse = new ListJobsResponse();
    if (listJobsResponse.next_page_token) {
      newListJobsResponse.next_page_token = listJobsResponse.next_page_token;
    }
    if (listJobsResponse.jobs && listJobsResponse.jobs.length > 0) {
      newListJobsResponse.jobs =
          listJobsResponse.jobs.map((p: any) => Job.buildFromObject(p));
    }
    return newListJobsResponse;
  }
}
