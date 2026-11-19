// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
import { vi } from 'vitest';

export function mockResizeObserver(width = 800, height = 600) {
  for (const [property, value] of Object.entries({
    offsetWidth: width,
    offsetHeight: height,
    clientWidth: width,
    clientHeight: height,
  })) {
    Object.defineProperty(HTMLElement.prototype, property, {
      configurable: true,
      get: () => value,
    });
  }
  class ResizeObserverMock {
    public disconnect = vi.fn();
    public unobserve = vi.fn();
    public observe: (element: Element) => void;
    constructor(callback: ResizeObserverCallback) {
      this.observe = vi.fn((element: Element) =>
        callback(
          [{ target: element, contentRect: { width, height } } as ResizeObserverEntry],
          this as unknown as ResizeObserver,
        ),
      );
    }
  }
  window.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver;
}
