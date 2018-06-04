import { JobMetadata } from './job';

export interface ListJobsResponse {
  jobs?: JobMetadata[];
  nextPageToken?: string;
}
