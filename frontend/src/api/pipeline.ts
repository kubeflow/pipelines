import { Parameter } from './parameter';

export class CronSchedule {
  private start_time?: string;
  private end_time?: string;
  private cron = '';

  constructor(cron: string, startTime?: string, endTime?: string) {
    this.cron = cron;
    if (startTime) {
      this.start_time = startTime;
    }
    if (endTime) {
      this.end_time = endTime;
    }
  }

  public static buildFromObject(cronSchedule: any): CronSchedule {
    const newCronSchedule = new CronSchedule(cronSchedule.cron);
    if (cronSchedule.start_time) {
      newCronSchedule.start_time = cronSchedule.start_time;
    }
    if (cronSchedule.end_time) {
      newCronSchedule.end_time = cronSchedule.end_time;
    }
    return newCronSchedule;
  }

  public get crontab(): string {
    return this.cron;
  }

  public get startTime(): string {
    return this.start_time || '';
  }

  public get endTime(): string {
    return this.end_time || '';
  }

  public toString(): string {
    return this.cron;
  }
}

export class PeriodicSchedule {
  private start_time?: string;
  private end_time?: string;
  private interval_second: number;

  constructor(intervalSeconds: number, startTime?: string, endTime?: string) {
    this.interval_second = intervalSeconds;
    if (startTime) {
      this.start_time = startTime;
    }
    if (endTime) {
      this.end_time = endTime;
    }
  }

  public static buildFromObject(periodicSchedule: any): PeriodicSchedule {
    const newPeriodicSchedule = new PeriodicSchedule(periodicSchedule.interval_second);
    if (periodicSchedule.start_time) {
      newPeriodicSchedule.start_time = periodicSchedule.start_time;
    }
    if (periodicSchedule.end_time) {
      newPeriodicSchedule.end_time = periodicSchedule.end_time;
    }
    return newPeriodicSchedule;
  }

  public get intervalSeconds(): number {
    return this.interval_second;
  }

  public get startTime(): string {
    return this.start_time || '';
  }

  public get endTime(): string {
    return this.end_time || '';
  }

  public toString(): string {
    const secInMin = 60;
    const secInHour = secInMin * 60;
    const secInDay = secInHour * 24;
    const secInWeek = secInDay * 7;
    const secInMonth = secInDay * 30;
    const months = Math.floor(this.interval_second / secInMonth);
    const weeks = Math.floor((this.interval_second % secInMonth) / secInWeek);
    const days = Math.floor((this.interval_second % secInWeek) / secInDay);
    const hours = Math.floor((this.interval_second % secInDay) / secInHour);
    const minutes = Math.floor((this.interval_second % secInHour) / secInMin);
    const seconds = Math.floor(this.interval_second % secInMin);
    let interval = 'Run every';
    if (months) {
      interval += ` ${months} months`;
    }
    if (weeks) {
      interval += ` ${weeks} weeks`;
    }
    if (days) {
      interval += ` ${days} days`;
    }
    if (hours) {
      interval += ` ${hours} hours`;
    }
    if (minutes) {
      interval += ` ${minutes} minutes`;
    }
    if (seconds) {
      interval += ` ${seconds} seconds`;
    }
    return interval;
  }
}

export class Trigger {
  private cron_schedule?: CronSchedule;
  private periodic_schedule?: PeriodicSchedule;

  constructor(schedule?: CronSchedule|PeriodicSchedule) {
    if (schedule instanceof CronSchedule) {
      this.cron_schedule = schedule;
    }
    if (schedule instanceof PeriodicSchedule) {
      this.periodic_schedule = schedule;
    }
  }

  public static buildFromObject(trigger: any): Trigger {
    const newTrigger = new Trigger();
    if (trigger.cron_schedule) {
      newTrigger.cron_schedule = CronSchedule.buildFromObject(trigger.cron_schedule);
    }
    if (trigger.periodic_schedule) {
      newTrigger.periodic_schedule = PeriodicSchedule.buildFromObject(trigger.periodic_schedule);
    }
    return newTrigger;
  }

  public get crontab(): string {
    return this.cron_schedule ? this.cron_schedule.crontab : '';
  }

  public get periodInSeconds(): number {
    return this.periodic_schedule ? this.periodic_schedule.intervalSeconds : -1;
  }

  public toString(): string {
    if (this.cron_schedule) {
      return this.cron_schedule.toString();
    }
    if (this.periodic_schedule) {
      this.periodic_schedule.toString();
    }
    return '';
  }
}

export class Pipeline {
  public id: string;
  public name: string;
  public description: string;
  public package_id: number;
  public enabled: boolean;
  // The status is surfacing the CRD's condition. A CRD can potentially have
  // multiple conditions, although in most cases, it should be in one state.
  // https://github.com/eBay/Kubernetes/blob/master/docs/devel/api-conventions.md
  // In multi-conditions case, the status is separated by semicolon.
  // STATUS_1;STATUS_2
  public status: string;
  public max_concurrency: number;
  public parameters: Parameter[];
  public trigger?: Trigger;
  public created_at: string;
  public updated_at: string;

  constructor() {
    this.name = '';
    this.description = '';
    this.package_id = -1;
    this.enabled = false;
    this.max_concurrency = 10;
    this.parameters = [];
  }

  public static buildFromObject(pipeline: any): Pipeline {
    const newPipeline = new Pipeline();
    newPipeline.id = pipeline.id;
    newPipeline.name = pipeline.name;
    newPipeline.description = pipeline.description;
    newPipeline.package_id = pipeline.package_id;
    newPipeline.enabled = pipeline.enabled;
    newPipeline.status = pipeline.status;
    newPipeline.max_concurrency = pipeline.max_concurrency;
    newPipeline.parameters = pipeline.parameters;
    if (pipeline.trigger) {
      newPipeline.trigger = Trigger.buildFromObject(pipeline.trigger);
    }
    newPipeline.created_at = pipeline.created_at;
    newPipeline.updated_at = pipeline.updated_at;
    return newPipeline;
  }
}
