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

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { LocalStorage, LocalStorageKey } from '../../lib/LocalStorage';

type Density = 'comfortable' | 'compact';
const DensityContext = createContext<{
  density: Density;
  setDensity: (density: Density) => void;
} | null>(null);

export function DensityProvider({ children }: { children: ReactNode }) {
  const [density, setDensityState] = useState(LocalStorage.getCollectionDensity);
  const setDensity = useCallback((value: Density) => {
    setDensityState(value);
    LocalStorage.saveCollectionDensity(value);
  }, []);

  // External sync: share the stored preference across browser tabs.
  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== LocalStorageKey.collectionDensity && event.key !== null) return;
      try {
        if (event.storageArea !== window.localStorage) return;
      } catch {
        return;
      }
      setDensityState(LocalStorage.getCollectionDensity());
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, []);

  const value = useMemo(() => ({ density, setDensity }), [density, setDensity]);
  return <DensityContext.Provider value={value}>{children}</DensityContext.Provider>;
}

/** Standalone tables use comfortable density when outside the application provider. */
export function useDensity() {
  return useContext(DensityContext);
}
