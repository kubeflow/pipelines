import { Pipeline } from './pipeline';

export class ListPipelinesResponse {
  public next_page_token?: string;
  public pipelines?: Pipeline[];

  public static buildFromObject(listPipelinesResponse: any): ListPipelinesResponse {
    const newListPipelinesResponse = new ListPipelinesResponse();
    if (listPipelinesResponse.next_page_token) {
      newListPipelinesResponse.next_page_token = listPipelinesResponse.next_page_token;
    }
    if (listPipelinesResponse.pipelines && listPipelinesResponse.pipelines.length > 0) {
      newListPipelinesResponse.pipelines =
          listPipelinesResponse.pipelines.map((p: any) => Pipeline.buildFromObject(p));
    }
    return newListPipelinesResponse;
  }
}
