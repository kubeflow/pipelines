import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

process.env.TZ = 'UTC';

// Pin toLocaleString to en-US so tests pass on any locale/machine
const _originalToLocaleString = Date.prototype.toLocaleString;
Date.prototype.toLocaleString = function(
  _locale?: string | string[],
  options?: Intl.DateTimeFormatOptions,
) {
  return _originalToLocaleString.call(this, 'en-US', options);
};

const _originalToLocaleDateString = Date.prototype.toLocaleDateString;
Date.prototype.toLocaleDateString = function(
  _locale?: string | string[],
  options?: Intl.DateTimeFormatOptions,
) {
  return _originalToLocaleDateString.call(this, 'en-US', options);
};

// @xyflow/react uses DOMMatrixReadOnly which jsdom does not implement.
if (!globalThis.DOMMatrixReadOnly) {
  // prettier-ignore
  class DOMMatrixReadOnlyMock {
    m11 = 1; m12 = 0; m13 = 0; m14 = 0;
    m21 = 0; m22 = 1; m23 = 0; m24 = 0;
    m31 = 0; m32 = 0; m33 = 1; m34 = 0;
    m41 = 0; m42 = 0; m43 = 0; m44 = 1;
    constructor(_init?: string) {}
  }
  globalThis.DOMMatrixReadOnly = DOMMatrixReadOnlyMock as unknown as typeof DOMMatrixReadOnly;
}

// jsdom does not default MouseEvent.view to window (jsdom/jsdom#3935, closed "not planned").
// d3-drag (used by @xyflow/react) accesses event.view.document on mousedown and mouseup,
// which throws when view is null. Patching the MouseEvent constructor (subclass, Proxy) or
// the UIEvent.prototype.view getter both break jsdom's internal WebIDL validation. Instead,
// capture-phase listeners patch view on already-constructed events before d3-drag reads it.
if (typeof window !== 'undefined') {
  const patchView = (event: MouseEvent) => {
    if (event.view === null) {
      try {
        (event as { view?: Window }).view = window;
      } catch {
        try {
          Object.defineProperty(event, 'view', {
            get: () => window,
            configurable: true,
          });
        } catch {
          // user-event v14 may create events with non-configurable view
        }
      }
    }
  };
  for (const type of ['mousedown', 'mouseup'] as const) {
    window.addEventListener(type, patchView, { capture: true });
  }
}

if (!globalThis.URL.createObjectURL) {
  globalThis.URL.createObjectURL = () => 'blob:mock';
}
if (!globalThis.URL.revokeObjectURL) {
  globalThis.URL.revokeObjectURL = () => {};
}

if (!globalThis.Worker) {
  class MockWorker {
    public onmessage: ((event: MessageEvent) => void) | null = null;
    public onerror: ((event: Event) => void) | null = null;

    public postMessage(): void {}

    public terminate(): void {}

    public addEventListener(): void {}

    public removeEventListener(): void {}

    public dispatchEvent(): boolean {
      return false;
    }
  }

  globalThis.Worker = MockWorker as unknown as typeof Worker;
}

const localStorageDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
if (!localStorageDescriptor || localStorageDescriptor.configurable) {
  const store = new Map<string, string>();
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: (key: string) => (store.has(key) ? store.get(key)! : null),
      setItem: (key: string, value: string) => {
        store.set(key, value);
      },
      removeItem: (key: string) => {
        store.delete(key);
      },
      clear: () => {
        store.clear();
      },
      key: (index: number) => Array.from(store.keys())[index] ?? null,
      get length() {
        return store.size;
      },
    } as Storage,
    configurable: true,
  });
}

afterEach(() => {
  cleanup();
});
