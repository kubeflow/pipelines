/*
 * Copyright 2018 The Kubeflow Authors
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

import { LocalStorage, LocalStorageKey } from './LocalStorage';

describe('LocalStorage', () => {
  let mockLocalStorage: { [key: string]: string | null };

  beforeEach(() => {
    // Mock localStorage
    mockLocalStorage = {};

    Storage.prototype.getItem = vi.fn((key: string) => mockLocalStorage[key] || null);
    Storage.prototype.setItem = vi.fn((key: string, value: string) => {
      mockLocalStorage[key] = value;
    });
    Storage.prototype.removeItem = vi.fn((key: string) => {
      delete mockLocalStorage[key];
    });
    Storage.prototype.clear = vi.fn(() => {
      mockLocalStorage = {};
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('hasKey', () => {
    it('returns false when key does not exist', () => {
      expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(false);
    });

    it('returns true when key exists with value "true"', () => {
      mockLocalStorage[LocalStorageKey.navbarCollapsed] = 'true';
      expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(true);
    });

    it('returns true when key exists with value "false"', () => {
      mockLocalStorage[LocalStorageKey.navbarCollapsed] = 'false';
      expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(true);
    });

    it('returns true when key exists with empty string value', () => {
      mockLocalStorage[LocalStorageKey.navbarCollapsed] = '';
      expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(true);
    });
  });

  describe('isNavbarCollapsed', () => {
    it('returns false when key does not exist', () => {
      expect(LocalStorage.isNavbarCollapsed()).toBe(false);
    });

    it('returns true when value is "true"', () => {
      mockLocalStorage[LocalStorageKey.navbarCollapsed] = 'true';
      expect(LocalStorage.isNavbarCollapsed()).toBe(true);
    });

    it('returns false when value is "false"', () => {
      mockLocalStorage[LocalStorageKey.navbarCollapsed] = 'false';
      expect(LocalStorage.isNavbarCollapsed()).toBe(false);
    });

    it('returns false when value is empty string', () => {
      mockLocalStorage[LocalStorageKey.navbarCollapsed] = '';
      expect(LocalStorage.isNavbarCollapsed()).toBe(false);
    });
  });

  describe('saveNavbarCollapsed', () => {
    it('saves true value as string "true"', () => {
      LocalStorage.saveNavbarCollapsed(true);
      expect(localStorage.setItem).toHaveBeenCalledWith(
        LocalStorageKey.navbarCollapsed,
        'true',
      );
    });

    it('saves false value as string "false"', () => {
      LocalStorage.saveNavbarCollapsed(false);
      expect(localStorage.setItem).toHaveBeenCalledWith(
        LocalStorageKey.navbarCollapsed,
        'false',
      );
    });
  });

  describe('integration: auto-collapse behavior', () => {
    it('manualCollapseState should be false when localStorage is empty (enables auto-collapse)', () => {
      // This test verifies the fix for issue #12801
      // When the key doesn't exist, hasKey should return false,
      // allowing SideNav to auto-collapse on mobile
      expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(false);
    });

    it('manualCollapseState should be true after user manually collapses nav', () => {
      LocalStorage.saveNavbarCollapsed(true);
      expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(true);
      expect(LocalStorage.isNavbarCollapsed()).toBe(true);
    });

    it('manualCollapseState should be true after user manually expands nav', () => {
      LocalStorage.saveNavbarCollapsed(false);
      expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(true);
      expect(LocalStorage.isNavbarCollapsed()).toBe(false);
    });
  });
});
