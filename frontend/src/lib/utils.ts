import { NODE_PHASE, NodePhase } from '../model/argo_template';

export function deleteAllChildren(parent: HTMLElement) {
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

export function listenOnce(element: Node, eventName: string, cb: Function) {
  const listener = (e: Event) => {
    e.target.removeEventListener(e.type, listener);
    return cb(e);
  };
  element.addEventListener(eventName, listener);
}

export function dateToString(date: number) {
  return date < 0 ? '-' : new Date(date).toLocaleString();
}

export function dateDiffToString(diff: number): string {
  const SECOND = 1000;
  const MINUTE = 60 * SECOND;
  const HOUR = 60 * MINUTE;
  let hours = 0;
  let minutes = 0;
  let seconds = 0;
  if (diff > HOUR) {
    hours = diff / 1000 / 60 / 60;
    diff -= hours * 1000 * 60 * 60;
  }
  if (diff > MINUTE) {
    minutes = diff / 1000 / 60;
    diff -= minutes * 60 * 60;
  }
  if (diff > SECOND) {
    seconds = diff / 1000;
  }
  return `${hours.toFixed()}:${minutes.toFixed()}:${seconds.toFixed()}`;
}

export function objectToArray(obj: {}) {
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

export function getAncestorElementWithClass(element: HTMLElement, className: string) {
  while (!element.classList.contains(className) && element.parentElement) {
    element = element.parentElement;
  }
  return element;
}

export function nodePhaseToIcon(status: NodePhase) {
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

export function nodePhaseToColor(status: NodePhase) {
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
