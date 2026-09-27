/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { Theme, ThemeProvider, useTheme } from './ThemeProvider';

class SystemThemeMedia extends EventTarget implements MediaQueryList {
  matches = false;
  media = '(prefers-color-scheme: dark)';
  onchange: ((event: MediaQueryListEvent) => void) | null = null;
  addListener = vi.fn();
  removeListener = vi.fn();

  setDark(dark: boolean) {
    this.matches = dark;
    this.dispatchEvent(new Event('change'));
  }
}

function ThemeControls() {
  const { theme, resolvedTheme, setTheme } = useTheme();
  return (
    <>
      <output aria-label='Selected theme'>{theme}</output>
      <output aria-label='Resolved theme'>{resolvedTheme}</output>
      {(['system', 'light', 'dark'] as Theme[]).map((option) => (
        <button key={option} onClick={() => setTheme(option)}>
          {option}
        </button>
      ))}
    </>
  );
}

function emitStorage(
  values: Partial<Pick<StorageEvent, 'key' | 'newValue' | 'storageArea' | 'url'>> = {},
) {
  // The test setup uses a plain Storage mock, which jsdom's StorageEvent rejects.
  const event = new Event('storage');
  const properties = {
    key: 'kfp.theme',
    newValue: null,
    storageArea: window.localStorage,
    url: window.location.href,
    ...values,
  };
  for (const [key, value] of Object.entries(properties)) {
    Object.defineProperty(event, key, { value });
  }
  window.dispatchEvent(event);
}

function expectTheme(theme: Theme, resolvedTheme: 'light' | 'dark') {
  expect(screen.getByLabelText('Selected theme')).toHaveTextContent(theme);
  expect(screen.getByLabelText('Resolved theme')).toHaveTextContent(resolvedTheme);
  const wrapper = screen.getByLabelText('Selected theme').closest('.kfp-theme');
  expect(wrapper).toHaveAttribute('data-theme', resolvedTheme);
  if (resolvedTheme === 'dark') {
    expect(wrapper).toHaveClass('dark');
  } else {
    expect(wrapper).not.toHaveClass('dark');
  }
}

describe('ThemeProvider', () => {
  let media: SystemThemeMedia;

  beforeEach(() => {
    localStorage.clear();
    media = new SystemThemeMedia();
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => media),
    );
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    localStorage.clear();
  });

  it('defaults to the system theme and follows live system changes', () => {
    render(
      <ThemeProvider className='preview'>
        <ThemeControls />
      </ThemeProvider>,
    );
    expectTheme('system', 'light');
    expect(screen.getByLabelText('Selected theme').closest('.kfp-theme')).toHaveClass('preview');
    act(() => media.setDark(true));
    expectTheme('system', 'dark');
    act(() => media.setDark(false));
    expectTheme('system', 'light');
    expect(localStorage.getItem('kfp.theme')).toBeNull();
  });

  it('resolves an initially dark system without changing the document root', () => {
    media.matches = true;
    const rootClass = document.documentElement.className;
    render(
      <ThemeProvider>
        <ThemeControls />
      </ThemeProvider>,
    );
    expectTheme('system', 'dark');
    expect(document.documentElement.className).toBe(rootClass);
  });

  it('persists explicit choices and ignores system changes until system is chosen', () => {
    render(
      <ThemeProvider>
        <ThemeControls />
      </ThemeProvider>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'dark' }));
    expectTheme('dark', 'dark');
    expect(localStorage.getItem('kfp.theme')).toBe('dark');
    act(() => media.setDark(false));
    expectTheme('dark', 'dark');
    fireEvent.click(screen.getByRole('button', { name: 'light' }));
    act(() => media.setDark(true));
    expectTheme('light', 'light');
    expect(localStorage.getItem('kfp.theme')).toBe('light');
    fireEvent.click(screen.getByRole('button', { name: 'system' }));
    expectTheme('system', 'dark');
    expect(localStorage.getItem('kfp.theme')).toBe('system');
  });

  it.each(['system', 'light', 'dark'] as Theme[])(
    'retains persisted %s through Strict Mode and remount without writing preferences',
    (savedTheme) => {
      localStorage.setItem('kfp.theme', savedTheme);
      localStorage.setItem('navbarCollapsed', 'true');
      localStorage.setItem('tablePageSize_runs', '50');
      const setItem = vi.spyOn(localStorage, 'setItem');
      const renderProvider = () =>
        render(
          <ThemeProvider>
            <ThemeControls />
          </ThemeProvider>,
        );
      const first = renderProvider();
      expectTheme(savedTheme, savedTheme === 'dark' ? 'dark' : 'light');
      first.unmount();
      renderProvider();
      expectTheme(savedTheme, savedTheme === 'dark' ? 'dark' : 'light');
      expect(setItem).not.toHaveBeenCalled();
      expect(localStorage.getItem('navbarCollapsed')).toBe('true');
      expect(localStorage.getItem('tablePageSize_runs')).toBe('50');
    },
  );

  it('uses an optional default only when storage is missing', () => {
    const first = render(
      <ThemeProvider defaultTheme='dark' storageKey='preview.theme'>
        <ThemeControls />
      </ThemeProvider>,
    );
    expectTheme('dark', 'dark');
    expect(localStorage.getItem('preview.theme')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'light' }));
    expect(localStorage.getItem('preview.theme')).toBe('light');
    expect(localStorage.getItem('kfp.theme')).toBeNull();
    first.unmount();
    render(
      <ThemeProvider defaultTheme='dark' storageKey='preview.theme'>
        <ThemeControls />
      </ThemeProvider>,
    );
    expectTheme('light', 'light');
  });

  it('falls back to system for invalid persisted values without overwriting them', () => {
    localStorage.setItem('kfp.theme', 'invalid');
    media.matches = true;
    render(
      <ThemeProvider defaultTheme='light'>
        <ThemeControls />
      </ThemeProvider>,
    );
    expectTheme('system', 'dark');
    expect(localStorage.getItem('kfp.theme')).toBe('invalid');
  });

  it('falls back to system when reading storage is denied', () => {
    vi.spyOn(localStorage, 'getItem').mockImplementation(() => {
      throw new DOMException('Storage access denied', 'SecurityError');
    });
    render(
      <ThemeProvider defaultTheme='dark'>
        <ThemeControls />
      </ThemeProvider>,
    );
    expectTheme('system', 'light');
  });

  it('keeps theme changes usable when writing storage is denied', () => {
    vi.spyOn(localStorage, 'setItem').mockImplementation(() => {
      throw new DOMException('Storage quota exceeded', 'QuotaExceededError');
    });
    render(
      <ThemeProvider>
        <ThemeControls />
      </ThemeProvider>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'dark' }));
    expectTheme('dark', 'dark');
    fireEvent.click(screen.getByRole('button', { name: 'light' }));
    expectTheme('light', 'light');
  });

  it('receives cross-tab updates and returns to system after removal or clear', () => {
    render(
      <ThemeProvider>
        <ThemeControls />
      </ThemeProvider>,
    );
    act(() => {
      localStorage.setItem('kfp.theme', 'dark');
      emitStorage({ newValue: 'dark' });
    });
    expectTheme('dark', 'dark');
    act(() => {
      localStorage.removeItem('kfp.theme');
      emitStorage();
    });
    expectTheme('system', 'light');
    fireEvent.click(screen.getByRole('button', { name: 'dark' }));
    act(() => {
      localStorage.clear();
      emitStorage({ key: null });
    });
    expectTheme('system', 'light');
  });

  it('reads current storage instead of applying an older queued event value', () => {
    render(
      <ThemeProvider>
        <ThemeControls />
      </ThemeProvider>,
    );
    act(() => {
      localStorage.setItem('kfp.theme', 'dark');
      emitStorage({ newValue: 'light' });
    });
    expectTheme('dark', 'dark');
    act(() => {
      localStorage.setItem('kfp.theme', 'invalid');
      emitStorage({ newValue: 'dark' });
    });
    expectTheme('system', 'light');
  });

  it('ignores unrelated, foreign-origin, and non-local storage events', () => {
    render(
      <ThemeProvider>
        <ThemeControls />
      </ThemeProvider>,
    );
    localStorage.setItem('kfp.theme', 'dark');
    act(() => {
      emitStorage({ key: 'navbarCollapsed', newValue: 'true' });
      emitStorage({ storageArea: window.sessionStorage, newValue: 'dark' });
      emitStorage({ storageArea: null, newValue: 'dark' });
      emitStorage({ url: 'https://example.invalid/', newValue: 'dark' });
      emitStorage({ url: '', newValue: 'dark' });
    });
    expectTheme('system', 'light');
  });

  it('removes media and storage listeners on unmount', () => {
    const addMedia = vi.spyOn(media, 'addEventListener');
    const removeMedia = vi.spyOn(media, 'removeEventListener');
    const addWindow = vi.spyOn(window, 'addEventListener');
    const removeWindow = vi.spyOn(window, 'removeEventListener');
    const view = render(
      <ThemeProvider>
        <ThemeControls />
      </ThemeProvider>,
    );
    view.unmount();
    for (const [event, listener] of addMedia.mock.calls) {
      expect(removeMedia).toHaveBeenCalledWith(event, listener);
    }
    for (const [, listener] of addWindow.mock.calls.filter(([event]) => event === 'storage')) {
      expect(removeWindow).toHaveBeenCalledWith('storage', listener);
    }
    expect(addMedia).toHaveBeenCalled();
    expect(addWindow.mock.calls.some(([event]) => event === 'storage')).toBe(true);
  });

  it('loads the new preference when the storage key changes', () => {
    localStorage.setItem('preview.light', 'light');
    localStorage.setItem('preview.dark', 'dark');
    const view = render(
      <ThemeProvider storageKey='preview.light'>
        <ThemeControls />
      </ThemeProvider>,
    );
    expectTheme('light', 'light');
    view.rerender(
      <ThemeProvider storageKey='preview.dark'>
        <ThemeControls />
      </ThemeProvider>,
    );
    expectTheme('dark', 'dark');
  });
});
