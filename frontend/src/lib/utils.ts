import { JobStatus } from '../model/job';

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

export function jobStatusToIcon(status: JobStatus) {
  switch (status) {
    case 'Running': return 'device:access-time';
    case 'Succeeded': return 'check-circle';
    case 'Skipped': return 'av:skip-next';
    case 'Failed': return 'av:skip-next';
    case 'Error': return 'error-outline';
    default:
      log.error('Unknown status:', status);
      return '';
  }
}
