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

export function objectToArray(obj: {}): any[] {
  if (!obj) {
    return [];
  }
  return Object.keys(obj).map((k) => {
    return {
      name: k,
      value: (obj as any)[k],
    };
  });
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
    case NODE_PHASE.SUCCEEDED: return '--success-color';
    case NODE_PHASE.ERROR: return '--error-color';
    case NODE_PHASE.FAILED: return '--error-color';
    case NODE_PHASE.RUNNING: return '--progress-color';
    case NODE_PHASE.SKIPPED: return '--neutral-color';
    default:
      log.error('Unknown status:', status);
      return '--neutral-color';
  }
}

export function showDialog(message: string): void {
  const dialog = document.createElement('message-dialog') as MessageDialog;
  document.body.appendChild(dialog);
  dialog.addEventListener('iron-overlay-closed', () => {
    document.body.removeChild(dialog);
  });

  dialog.message = message;
  dialog.open();
}

export function showNotification(message: string): void {
  const toast = document.createElement('paper-toast');
  document.body.appendChild(toast);
  toast.addEventListener('iron-overlay-closed', () => {
    document.body.removeChild(toast);
  });

  toast.text = message;
  toast.open();
}
