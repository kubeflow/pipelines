import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

process.env.TZ = 'UTC';

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

  globalThis.Worker = (MockWorker as unknown) as typeof Worker;
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
