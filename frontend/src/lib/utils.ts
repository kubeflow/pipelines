import { Parameter, SweepValue } from "src/lib/parameter";

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

export function isSweepParameter(param: Parameter) {
  const sweepTest = param.value as SweepValue;
  return sweepTest.from !== undefined &&
    sweepTest.to !== undefined &&
    sweepTest.step !== undefined;
}

export function dateDiffToString(diff: number): string {
  const SECOND = 1000, MINUTE = 60 * SECOND, HOUR = 60 * MINUTE;
  let hours = 0, minutes = 0, seconds = 0;
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