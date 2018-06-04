import { PipelinePackage } from './pipeline_package';

export interface ListPackagesResponse {
  nextPageToken?: string;
  packages?: PipelinePackage[];
}
