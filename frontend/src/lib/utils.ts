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

import { DialogResult, PopupDialog } from '../components/popup-dialog/popup-dialog';

import 'paper-toast/paper-toast';
import { Trigger } from '../api/job';
import '../components/popup-dialog/popup-dialog';

export const NODE_PHASE = {
  ERROR: 'Error',
  FAILED: 'Failed',
  RUNNING: 'Running',
  SKIPPED: 'Skipped',
  SUCCEEDED: 'Succeeded',
};

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

export function enabledDisplayString(trigger: Trigger|undefined, enabled: boolean): string {
  if (trigger) {
    return enabled ? 'Yes' : 'No';
  }
  return '-';
}

export function formatDateString(date: string | undefined): string {
  return date ? new Date(date).toLocaleString() : '-';
}

export function getRunTime(start: string, end: string, status: any): string {
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

export function nodePhaseToIcon(status: any|string): string {
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

export function nodePhaseToColor(status: any): string {
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
