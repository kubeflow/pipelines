// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

  public get cronExpression(): string {
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
      interval += ` ${months} months,`;
    }
    if (weeks) {
      interval += ` ${weeks} weeks,`;
    }
    if (days) {
      interval += ` ${days} days,`;
    }
    if (hours) {
      interval += ` ${hours} hours,`;
    }
    if (minutes) {
      interval += ` ${minutes} minutes,`;
    }
    if (seconds) {
      interval += ` ${seconds} seconds,`;
    }
    // Add 'and' if necessary
    const insertAndLocation = interval.lastIndexOf(', ') + 1;
    if (insertAndLocation > 0) {
      interval = interval.slice(0, insertAndLocation) + ' and' + interval.slice(insertAndLocation);
    }
    // Remove trailing comma
    return interval.slice(0, -1);
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

  public get cronExpression(): string {
    return this.cron_schedule ? this.cron_schedule.cronExpression : '';
  }

  public get periodInSeconds(): number {
    return this.periodic_schedule ? this.periodic_schedule.intervalSeconds : -1;
  }

  public toString(): string {
    if (this.cron_schedule) {
      return this.cron_schedule.toString();
    }
    if (this.periodic_schedule) {
      return this.periodic_schedule.toString();
    }
    return '';
  }
}

export class Job {
  public id: string;
  public name: string;
  public description: string;
  public pipeline_id: string;
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
  public created_at: string | undefined;
  public updated_at: string | undefined;

  constructor() {
    this.id = '';
    this.name = '';
    this.description = '';
    this.pipeline_id = '';
    this.enabled = false;
    this.max_concurrency = 10;
    this.parameters = [];
    this.status = '';
    this.created_at = undefined;
    this.updated_at = undefined;
  }

  public static buildFromObject(job: any): Job {
    const newJob = new Job();
    newJob.id = job.id;
    newJob.name = job.name;
    newJob.description = job.description;
    newJob.pipeline_id = job.pipeline_id;
    newJob.enabled = job.enabled;
    newJob.status = job.status;
    newJob.max_concurrency = job.max_concurrency;
    newJob.parameters = job.parameters;
    if (job.trigger) {
      newJob.trigger = Trigger.buildFromObject(job.trigger);
    }
    newJob.created_at = job.created_at;
    newJob.updated_at = job.updated_at;
    return newJob;
  }
}
