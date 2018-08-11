import { Pipeline } from './pipeline';

export interface ListPipelinesResponse {
  next_page_token?: string;
  pipelines?: Pipeline[];
}
