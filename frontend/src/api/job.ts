export class JobMetadata {
  public id: string;
  public name: string;
  public namespace: string;
  public created_at: string;
  public scheduled_at: string;
  public status: string;

  public static buildFromObject(job: any): JobMetadata {
    const newJobMetadata = new JobMetadata();
    newJobMetadata.id = job.id;
    newJobMetadata.name = job.name;
    newJobMetadata.namespace = job.namespace;
    newJobMetadata.created_at = job.created_at;
    newJobMetadata.scheduled_at = job.scheduled_at;
    newJobMetadata.status = job.status;
    return newJobMetadata;
  }
}

export class Job {
  public job: JobMetadata;
  public workflow: string;

  public static buildFromObject(oldJob: any): Job {
    const newJob = new Job();
    newJob.job = JobMetadata.buildFromObject(oldJob.job);
    newJob.workflow = oldJob.workflow;
    return newJob;
  }
}
