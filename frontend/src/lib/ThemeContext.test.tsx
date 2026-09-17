/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { AppThemeProvider, useThemeContext } from './ThemeContext';
import { LocalStorage, LocalStorageKey } from './LocalStorage';

const TestComponent = () => {
  const { activeMode, themeMode, toggleTheme, setThemeMode } = useThemeContext();
  return (
    <div>
      <span data-testid="active-mode">{activeMode}</span>
      <span data-testid="theme-mode">{themeMode}</span>
      <button data-testid="toggle-btn" onClick={toggleTheme}>
        Toggle
      </button>
      <button data-testid="dark-btn" onClick={() => setThemeMode('dark')}>
        Dark
      </button>
      <button data-testid="light-btn" onClick={() => setThemeMode('light')}>
        Light
      </button>
    </div>
  );
};

describe('ThemeContext', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('defaults to system theme mode when local storage is empty', () => {
    render(
      <AppThemeProvider>
        <TestComponent />
      </AppThemeProvider>,
    );

    expect(screen.getByTestId('theme-mode').textContent).toBe('system');
  });

  it('toggles theme between light and dark and persists in localStorage', () => {
    render(
      <AppThemeProvider>
        <TestComponent />
      </AppThemeProvider>,
    );

    const toggleBtn = screen.getByTestId('toggle-btn');
    fireEvent.click(toggleBtn);

    expect(screen.getByTestId('active-mode').textContent).toBe('dark');
    expect(LocalStorage.getThemeMode()).toBe('dark');

    fireEvent.click(toggleBtn);
    expect(screen.getByTestId('active-mode').textContent).toBe('light');
    expect(LocalStorage.getThemeMode()).toBe('light');
  });

  it('explicitly sets dark mode and updates document data-theme attribute', () => {
    render(
      <AppThemeProvider>
        <TestComponent />
      </AppThemeProvider>,
    );

    fireEvent.click(screen.getByTestId('dark-btn'));
    expect(screen.getByTestId('active-mode').textContent).toBe('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');

    fireEvent.click(screen.getByTestId('light-btn'));
    expect(screen.getByTestId('active-mode').textContent).toBe('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
  });
});
