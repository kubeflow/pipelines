import { PipelinePackage } from './pipeline_package';

export interface ListPackagesResponse {
  next_page_token?: string;
  packages?: PipelinePackage[];
}
