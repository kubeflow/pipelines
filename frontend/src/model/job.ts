import { Workflow } from '../model/argo_template';

export interface JobMetadata {
  id: number;
  createdAt: string;
  name: string;
  scheduledAt: number;
}

export interface Job {
  metadata: JobMetadata;
  jobDetail: Workflow;
}
