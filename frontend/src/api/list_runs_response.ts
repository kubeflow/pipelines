import { RunMetadata } from './run';

export class ListRunsResponse {
  public runs?: RunMetadata[];
  public next_page_token?: string;

  public static buildFromObject(listRunsResponse: any): ListRunsResponse {
    const newListRunsResponse = new ListRunsResponse();
    newListRunsResponse.next_page_token = listRunsResponse.next_page_token;
    if (listRunsResponse.runs && listRunsResponse.runs.length > 0) {
      newListRunsResponse.runs =
          listRunsResponse.runs.map((r: any) => RunMetadata.buildFromObject(r));
    }
    return newListRunsResponse;
  }
}
