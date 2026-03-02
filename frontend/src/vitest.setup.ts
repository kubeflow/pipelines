import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

process.env.TZ = 'UTC';

// @xyflow/react uses DOMMatrixReadOnly which jsdom does not implement.
if (!globalThis.DOMMatrixReadOnly) {
  class DOMMatrixReadOnlyMock {
    m22: number;
    constructor(_init?: string) {
      this.m22 = 1;
    }
  }
  globalThis.DOMMatrixReadOnly = (DOMMatrixReadOnlyMock as unknown) as typeof DOMMatrixReadOnly;
}

// @xyflow/react's d3-drag dependency accesses event.view.document on mousedown,
// but jsdom leaves event.view null. This must be global (not per-file beforeAll)
// because Vitest's process-level uncaught-exception monitor fires before scoped
// window error handlers in parallel worker threads.
if (typeof window !== 'undefined') {
  window.addEventListener('error', (event: ErrorEvent) => {
    if (
      event.error instanceof TypeError &&
      event.error.message.includes("Cannot read properties of null (reading 'document')")
    ) {
      event.preventDefault();
    }
  });
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
