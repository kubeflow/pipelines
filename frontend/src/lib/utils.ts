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

import { NODE_PHASE, NodePhase } from '../../third_party/argo-ui/argo_template';
import { DialogResult, PopupDialog } from '../components/popup-dialog/popup-dialog';

import 'paper-toast/paper-toast';
import { apiTrigger } from '../api/job';
import '../components/popup-dialog/popup-dialog';

export function deleteAllChildren(parent: HTMLElement): void {
  while (parent.firstChild) {
    parent.removeChild(parent.firstChild);
  }
}

export const log = {
  error: (...args: any[]) => {
    // tslint:disable-next-line:no-console
    console.error(...args);
  },
  verbose: (...args: any[]) => {
    // tslint:disable-next-line:no-console
    console.log(...args);
  },
};

export function listenOnce(element: Node, eventName: string, cb: Function): void {
  const listener = (e: Event) => {
    if (e.target) {
      e.target.removeEventListener(e.type, listener);
    }
    return cb(e);
  };
  element.addEventListener(eventName, listener);
}

export function enabledDisplayString(trigger: apiTrigger|undefined, enabled: boolean): string {
  if (trigger) {
    return enabled ? 'Yes' : 'No';
  }
  return '-';
}

export function formatDateString(date: Date | string | undefined): string {
  if (typeof date === 'string') {
    return new Date(date).toLocaleString();
  } else {
    return date ? date.toLocaleString() : '-';
  }
}

export function triggerDisplayString(trigger?: apiTrigger): string {
  if (trigger) {
    if (trigger.cron_schedule && trigger.cron_schedule.cron) {
      return trigger.cron_schedule.cron;
    }
    if (trigger.periodic_schedule && trigger.periodic_schedule.interval_second) {
      const intervalSecond = trigger.periodic_schedule.interval_second;
      const secInMin = 60;
      const secInHour = secInMin * 60;
      const secInDay = secInHour * 24;
      const secInWeek = secInDay * 7;
      const secInMonth = secInDay * 30;
      const months = Math.floor(+intervalSecond / secInMonth);
      const weeks = Math.floor((+intervalSecond % secInMonth) / secInWeek);
      const days = Math.floor((+intervalSecond % secInWeek) / secInDay);
      const hours = Math.floor((+intervalSecond % secInDay) / secInHour);
      const minutes = Math.floor((+intervalSecond % secInHour) / secInMin);
      const seconds = Math.floor(+intervalSecond % secInMin);
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
        interval = interval.slice(0, insertAndLocation) +
          ' and' + interval.slice(insertAndLocation);
      }
      // Remove trailing comma
      return interval.slice(0, -1);
    }
  }
  return '-';
}

export function getRunTime(start: string, end: string, status: NodePhase): string {
  if (!status) {
    return '-';
  }
  const parsedStart = start ? new Date(start).getTime() : 0;
  const parsedEnd = end ? new Date(end).getTime() : Date.now();

  return (parsedStart && parsedEnd) ? dateDiffToString(parsedEnd - parsedStart) : '-';
}

export function dateDiffToString(diff: number): string {
  const SECOND = 1000;
  const MINUTE = 60 * SECOND;
  const HOUR = 60 * MINUTE;
  const seconds = Math.floor((diff / SECOND) % 60).toString().padStart(2, '0');
  const minutes = Math.floor((diff / MINUTE) % 60).toString().padStart(2, '0');
  // Hours are the largest denomination, so we don't pad them
  const hours = Math.floor((diff / HOUR) % 24).toString();
  return `${hours}:${minutes}:${seconds}`;
}

export function getAncestorElementWithClass(element: HTMLElement, className: string): Element {
  while (!element.classList.contains(className) && element.parentElement) {
    element = element.parentElement;
  }
  return element;
}

export function nodePhaseToIcon(status: NodePhase|string): string {
  switch (status) {
    case NODE_PHASE.RUNNING: return 'device:access-time';
    case NODE_PHASE.SUCCEEDED: return 'check-circle';
    case NODE_PHASE.SKIPPED: return 'av:skip-next';
    case NODE_PHASE.FAILED: return 'error-outline';
    case NODE_PHASE.ERROR: return 'error-outline';
    case 'NONE': return 'av:fiber-manual-record';
    default:
      log.error('Unknown status:', status);
      return 'help-outline';
  }
}

export function nodePhaseToColor(status: NodePhase): string {
  switch (status) {
    case NODE_PHASE.SUCCEEDED: return '--good-color';
    case NODE_PHASE.ERROR: return '--danger-color';
    case NODE_PHASE.FAILED: return '--danger-color';
    case NODE_PHASE.RUNNING: return '--theme-primary-color';
    case NODE_PHASE.SKIPPED: return '--inactive-color';
    default:
      log.error('Unknown status:', status);
      return '--inactive-color';
  }
}

export function showDialog(
    title: string,
    body?: string,
    button1?: string,
    button2?: string): Promise<DialogResult> {
  const dialog = document.createElement('popup-dialog') as PopupDialog;
  document.body.appendChild(dialog);
  dialog.addEventListener('iron-overlay-closed', () => document.body.removeChild(dialog));

  dialog.title = title;
  dialog.body = body !== undefined ? body : '';
  dialog.button1 = button1 !== undefined ? button1 : '';
  dialog.button2 = button2 !== undefined ? button2 : '';

  return dialog.open();
}

export function showNotification(message: string): void {
  const toast = document.createElement('paper-toast');
  toast.duration = 5000;
  document.body.appendChild(toast);
  toast.addEventListener('iron-overlay-closed', () => {
    document.body.removeChild(toast);
  });

  toast.text = message;
  toast.open();
}

export function wait(timeMs: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve();
    }, timeMs);
  });
}
