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

import '@fontsource/public-sans/latin-400.css';
import '@fontsource/public-sans/latin-500.css';
import '@fontsource/public-sans/latin-600.css';
import '@fontsource/jetbrains-mono/latin-400.css';
import '@fontsource/jetbrains-mono/latin-500.css';

import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
} from 'react';

export type Theme = 'system' | 'light' | 'dark';
type ResolvedTheme = Exclude<Theme, 'system'>;

interface ThemeContextValue {
  theme: Theme;
  resolvedTheme: ResolvedTheme;
  setTheme: (theme: Theme) => void;
}

interface ThemeProviderProps {
  children: React.ReactNode;
  className?: string;
  defaultTheme?: Theme;
  storageKey?: string;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);
const SYSTEM_THEME_QUERY = '(prefers-color-scheme: dark)';

function readTheme(storageKey: string, missingTheme: Theme): Theme {
  try {
    const savedTheme = window.localStorage.getItem(storageKey);
    if (savedTheme === null) return missingTheme;
    return savedTheme === 'light' || savedTheme === 'dark' ? savedTheme : 'system';
  } catch {
    return 'system';
  }
}

function subscribeToSystemTheme(onChange: () => void): () => void {
  const media = window.matchMedia(SYSTEM_THEME_QUERY);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}

function getSystemTheme(): ResolvedTheme {
  return window.matchMedia(SYSTEM_THEME_QUERY).matches ? 'dark' : 'light';
}

export function ThemeProvider({
  children,
  className,
  defaultTheme = 'system',
  storageKey = 'kfp.theme',
}: ThemeProviderProps) {
  return (
    <ThemeProviderContent
      key={storageKey}
      className={className}
      defaultTheme={defaultTheme}
      storageKey={storageKey}
    >
      {children}
    </ThemeProviderContent>
  );
}

function ThemeProviderContent({
  children,
  className,
  defaultTheme,
  storageKey,
}: Required<Pick<ThemeProviderProps, 'defaultTheme' | 'storageKey'>> & ThemeProviderProps) {
  const [theme, setThemeState] = useState(() => readTheme(storageKey, defaultTheme));
  const systemTheme = useSyncExternalStore(subscribeToSystemTheme, getSystemTheme);
  const resolvedTheme = theme === 'system' ? systemTheme : theme;

  const setTheme = useCallback(
    (nextTheme: Theme) => {
      setThemeState(nextTheme);
      try {
        window.localStorage.setItem(storageKey, nextTheme);
      } catch {
        // The chosen theme remains usable for this session when storage is unavailable.
      }
    },
    [storageKey],
  );

  // External sync: receive preference changes from other tabs without writing them back.
  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== storageKey && event.key !== null) return;
      try {
        if (
          event.storageArea !== window.localStorage ||
          new URL(event.url).origin !== window.location.origin
        ) {
          return;
        }
      } catch {
        return;
      }
      // Read current storage so an older queued event cannot restore a stale preference.
      setThemeState(readTheme(storageKey, 'system'));
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, [storageKey]);

  const value = useMemo(
    () => ({ theme, resolvedTheme, setTheme }),
    [theme, resolvedTheme, setTheme],
  );
  return (
    <ThemeContext.Provider value={value}>
      <div
        className={['kfp-theme', resolvedTheme === 'dark' && 'dark', className]
          .filter(Boolean)
          .join(' ')}
        data-theme={resolvedTheme}
      >
        {children}
      </div>
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const theme = useContext(ThemeContext);
  if (!theme) throw new Error('useTheme must be used within a ThemeProvider.');
  return theme;
}
