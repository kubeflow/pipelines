import { DialogResult, PopupDialog } from '../components/popup-dialog/popup-dialog';
import { NODE_PHASE, NodePhase } from '../model/argo_template';

import 'paper-toast/paper-toast';
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
    e.target.removeEventListener(e.type, listener);
    return cb(e);
  };
  element.addEventListener(eventName, listener);
}

export function enabledDisplayString(schedule: string, enabled: boolean): string {
  if (schedule) {
    return enabled ? 'Yes' : 'No';
  }
  return '-';
}

export function formatDateString(date: string): string {
  return date ? new Date(date).toLocaleString() : '-';
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
  const seconds = Math.floor((diff / SECOND) % 60);
  const minutes = Math.floor((diff / MINUTE) % 60);
  const hours = Math.floor((diff / HOUR) % 24);
  return `${hours.toFixed()}:${minutes.toFixed()}:${seconds.toFixed()}`;
}

export function getAncestorElementWithClass(element: HTMLElement, className: string): Element {
  while (!element.classList.contains(className) && element.parentElement) {
    element = element.parentElement;
  }
  return element;
}

export function nodePhaseToIcon(status: NodePhase): string {
  switch (status) {
    case NODE_PHASE.RUNNING: return 'device:access-time';
    case NODE_PHASE.SUCCEEDED: return 'check-circle';
    case NODE_PHASE.SKIPPED: return 'av:skip-next';
    case NODE_PHASE.FAILED: return 'error-outline';
    case NODE_PHASE.ERROR: return 'error-outline';
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
