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

export enum LocalStorageKey {
  navbarCollapsed = 'navbarCollapsed',
  tablePageSize = 'tablePageSize',
}

export class LocalStorage {
  public static hasKey(key: LocalStorageKey): boolean {
    return localStorage.getItem(key) !== null;
  }

  public static isNavbarCollapsed(): boolean {
    try {
      return localStorage.getItem(LocalStorageKey.navbarCollapsed) === 'true';
    } catch {
      return false;
    }
  }

  public static saveNavbarCollapsed(value: boolean): void {
    try {
      localStorage.setItem(LocalStorageKey.navbarCollapsed, value.toString());
    } catch {
      // Navigation remains usable when browser policy or quota blocks storage.
    }
  }

  public static getTablePageSize(pageId?: string): number {
    const key = pageId
      ? `${LocalStorageKey.tablePageSize}_${pageId}`
      : LocalStorageKey.tablePageSize;
    try {
      const parsed = Number(localStorage.getItem(key));
      return [10, 20, 50, 100].includes(parsed) ? parsed : 10;
    } catch {
      return 10;
    }
  }

  public static saveTablePageSize(value: number, pageId?: string): void {
    const normalizedValue = [10, 20, 50, 100].includes(value) ? value : 10;
    const key = pageId
      ? `${LocalStorageKey.tablePageSize}_${pageId}`
      : LocalStorageKey.tablePageSize;
    try {
      localStorage.setItem(key, normalizedValue.toString());
    } catch {
      // Paging remains usable when browser policy or quota blocks storage.
    }
  }
}
