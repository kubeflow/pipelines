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
  pipelineView = 'pipelineView',
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

  public static getPipelineView(): 'table' | 'cards' {
    try {
      return localStorage.getItem(LocalStorageKey.pipelineView) === 'cards' ? 'cards' : 'table';
    } catch {
      return 'table';
    }
  }

  public static savePipelineView(value: 'table' | 'cards'): void {
    try {
      localStorage.setItem(LocalStorageKey.pipelineView, value);
    } catch {
      // The view can still change when browser policy or quota blocks storage.
    }
  }

  public static getTableColumnWidths(tableId: string): number[] | undefined {
    try {
      const widths: unknown = JSON.parse(
        localStorage.getItem(`tableColumnWidths_${tableId}`) || 'null',
      );
      return Array.isArray(widths) &&
        widths.length > 0 &&
        widths.every(
          (width) =>
            typeof width === 'number' && Number.isFinite(width) && width >= 80 && width <= 2000,
        )
        ? widths
        : undefined;
    } catch {
      return undefined;
    }
  }

  public static getTableFlexibleColumn(tableId: string): number | undefined {
    try {
      const stored = localStorage.getItem(`tableFlexibleColumn_${tableId}`);
      const index = stored === null ? NaN : Number(stored);
      return Number.isInteger(index) && index >= 0 ? index : undefined;
    } catch {
      return undefined;
    }
  }

  public static saveTableColumnWidths(
    tableId: string,
    widths?: number[],
    flexibleColumn?: number,
  ): void {
    try {
      const key = `tableColumnWidths_${tableId}`;
      if (widths) localStorage.setItem(key, JSON.stringify(widths));
      else localStorage.removeItem(key);
      const flexibleKey = `tableFlexibleColumn_${tableId}`;
      if (widths && flexibleColumn !== undefined)
        localStorage.setItem(flexibleKey, String(flexibleColumn));
      else localStorage.removeItem(flexibleKey);
    } catch {
      // Resizing remains available when browser storage is blocked.
    }
  }

  public static getTablePageSize(pageId?: string): number {
    const key = pageId
      ? `${LocalStorageKey.tablePageSize}_${pageId}`
      : LocalStorageKey.tablePageSize;
    try {
      const parsed = Number(localStorage.getItem(key));
      return [10, 20, 50, 100].includes(parsed) ? parsed : 20;
    } catch {
      return 20;
    }
  }

  public static saveTablePageSize(value: number, pageId?: string): void {
    const normalizedValue = [10, 20, 50, 100].includes(value) ? value : 20;
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
