export class RunMetadata {
  public id = '';
  public name = '';
  public namespace = '';
  public created_at = '';
  public scheduled_at = '';
  public status = '';

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
  public run = RunMetadata.buildFromObject({});
  public workflow = '';

  public static buildFromObject(oldRun: any): Run {
    const newRun = new Run();
    newRun.run = RunMetadata.buildFromObject(oldRun.run);
    newRun.workflow = oldRun.workflow;
    return newRun;
  }
}
