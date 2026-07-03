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
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('returns false for hasKey when key does not exist', () => {
    expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(false);
  });

  it('returns true for hasKey when key exists', () => {
    localStorage.setItem(LocalStorageKey.navbarCollapsed, 'false');
    expect(LocalStorage.hasKey(LocalStorageKey.navbarCollapsed)).toBe(true);
  });

  it('reads and writes navbar collapsed state', () => {
    LocalStorage.saveNavbarCollapsed(true);
    expect(LocalStorage.isNavbarCollapsed()).toBe(true);
  });

  it('returns default page size of 10 when no value is saved', () => {
    expect(LocalStorage.getTablePageSize()).toBe(10);
  });

  it('reads and writes table page size', () => {
    LocalStorage.saveTablePageSize(50);
    expect(LocalStorage.getTablePageSize()).toBe(50);
  });

  it('returns default page size for invalid stored values', () => {
    localStorage.setItem(LocalStorageKey.tablePageSize, '25');
    expect(LocalStorage.getTablePageSize()).toBe(10);
  });

  it('accepts all valid page size options', () => {
    for (const size of [10, 20, 50, 100]) {
      LocalStorage.saveTablePageSize(size);
      expect(LocalStorage.getTablePageSize()).toBe(size);
    }
  });

  it('stores page size per page independently', () => {
    LocalStorage.saveTablePageSize(50, 'runs');
    LocalStorage.saveTablePageSize(20, 'experiments');
    expect(LocalStorage.getTablePageSize('runs')).toBe(50);
    expect(LocalStorage.getTablePageSize('experiments')).toBe(20);
  });

  it('returns default for a page with no saved value', () => {
    LocalStorage.saveTablePageSize(50, 'runs');
    expect(LocalStorage.getTablePageSize('pipelines')).toBe(10);
  });
});
