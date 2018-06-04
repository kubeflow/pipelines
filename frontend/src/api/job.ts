export interface JobMetadata {
  created_at: string;
  name: string;
  scheduled_at: string;
}

export interface Job {
  job: JobMetadata;
  workflow: string;
}
