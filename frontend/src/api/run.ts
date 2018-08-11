export class RunMetadata {
  public id: string;
  public name: string;
  public namespace: string;
  public created_at: string;
  public scheduled_at: string;
  public status: string;

  public static buildFromObject(run: any): RunMetadata {
    const newRunMetadata = new RunMetadata();
    newRunMetadata.id = run.id;
    newRunMetadata.name = run.name;
    newRunMetadata.namespace = run.namespace;
    newRunMetadata.created_at = run.created_at;
    newRunMetadata.scheduled_at = run.scheduled_at;
    newRunMetadata.status = run.status;
    return newRunMetadata;
  }
}

export class Run {
  public run: RunMetadata;
  public workflow: string;

  public static buildFromObject(oldRun: any): Run {
    const newRun = new Run();
    newRun.run = RunMetadata.buildFromObject(oldRun.run);
    newRun.workflow = oldRun.workflow;
    return newRun;
  }
}
