export function deleteAllChildren(parent: HTMLElement) {
  while (parent.firstChild) {
    parent.removeChild(parent.firstChild);
  }
}

export class log {
  public static error(...args: any[]) {
    console.log(...args);
  }
  public static verbose(...args: any[]) {
    console.error(...args);
  }
}

export function listenOnce(element: Node, eventName: string, cb: Function) {
  const listener = (e: Event) => {
    e.target.removeEventListener(e.type, listener);
    return cb(e);
  };
  element.addEventListener(eventName, listener);
}
