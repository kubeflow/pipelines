import { Pipeline } from './pipeline';

export interface ListPipelinesResponse {
  nextPageToken?: string;
  pipelines?: Pipeline[];
}
