import { Workflow } from '../model/argo_template';

export interface JobMetadata {
  id: number;
  // createdAt is SECONDS since epoch
  createdAt: number;
  name: string;
  // createdAt is SECONDS since epoch
  scheduledAt: number;
}

export interface Job {
  metadata: JobMetadata;
  jobDetail: Workflow;
}
