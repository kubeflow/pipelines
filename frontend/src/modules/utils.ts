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