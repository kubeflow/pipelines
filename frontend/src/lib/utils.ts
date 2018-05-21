import { MessageDialog } from '../components/message-dialog/message-dialog';
import { NODE_PHASE, NodePhase } from '../model/argo_template';

import 'paper-toast/paper-toast';
import '../components/message-dialog/message-dialog';

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

export function formatDateString(date: string): string {
  return date ? new Date(date).toLocaleString() : '-';
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

export function showDialog(message: string, details?: string): void {
  const dialog = document.createElement('message-dialog') as MessageDialog;
  document.body.appendChild(dialog);
  dialog.addEventListener('iron-overlay-closed', () => {
    document.body.removeChild(dialog);
  });

  dialog.message = message;
  if (details) {
    dialog.details = details;
  }
  dialog.open();
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
