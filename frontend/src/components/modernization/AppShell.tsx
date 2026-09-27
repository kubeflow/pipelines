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

import { useId, useRef, useState, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import {
  BookOpen,
  ChevronLeft,
  ChevronRight,
  CodeXml,
  MessageSquare,
  Monitor,
  Moon,
  Sun,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import KubeflowLogo from 'src/icons/kubeflowLogo';
import { LocalStorageKey } from 'src/lib/LocalStorage';
import { Button } from '../ui/button';
import { useTheme } from './ThemeProvider';
import './AppShell.css';

export interface AppShellNavItem {
  id: string;
  label: string;
  href: string;
  icon: LucideIcon;
  active?: boolean;
}

export interface AppShellProps {
  children: ReactNode;
  items: readonly AppShellNavItem[];
  currentPath: string;
  secondaryItems?: readonly AppShellNavItem[];
  hideSideNav?: boolean;
  namespace?: string;
  version?: string;
  versionHref?: string;
  metadata?: {
    clusterName?: string;
    clusterHref?: string;
    projectId?: string;
    buildDate?: string;
    commitHash?: string;
  };
}

function readCollapsedPreference(): boolean {
  try {
    return localStorage.getItem(LocalStorageKey.navbarCollapsed) === 'true';
  } catch {
    return false;
  }
}

function subscribeToViewport(onChange: () => void) {
  window.addEventListener('resize', onChange);
  return () => window.removeEventListener('resize', onChange);
}

function isNarrowViewport() {
  return window.innerWidth < 1024;
}

const footerLinks = [
  {
    label: 'Documentation',
    href: 'https://www.kubeflow.org/docs/pipelines/',
    icon: BookOpen,
  },
  { label: 'GitHub', href: 'https://github.com/kubeflow/pipelines', icon: CodeXml },
  {
    label: 'Report an issue',
    href: 'https://github.com/kubeflow/pipelines/issues/new/choose',
    icon: MessageSquare,
  },
];

export function AppShell({
  children,
  items,
  currentPath,
  secondaryItems = [],
  hideSideNav = false,
  namespace,
  version,
  versionHref,
  metadata,
}: AppShellProps) {
  const [preferredCollapsed, setPreferredCollapsed] = useState(readCollapsedPreference);
  const isNarrow = useSyncExternalStore(subscribeToViewport, isNarrowViewport, () => false);
  const collapsed = isNarrow || preferredCollapsed;
  const { theme, setTheme } = useTheme();
  const mainId = useId();
  const mainRef = useRef<HTMLElement>(null);
  const themeId = useId();
  const ThemeIcon = theme === 'system' ? Monitor : theme === 'dark' ? Moon : Sun;

  function toggleCollapsed() {
    const next = !preferredCollapsed;
    setPreferredCollapsed(next);
    try {
      localStorage.setItem(LocalStorageKey.navbarCollapsed, String(next));
    } catch {
      // Navigation stays usable when browser policy or quota prevents persistence.
    }
  }

  function renderNavigationItem(item: AppShellNavItem) {
    const Icon = item.icon;
    const active = item.active ?? currentPath === item.href;
    return (
      <li key={item.id}>
        <Link
          to={item.href}
          className='kfp-shell-nav-link'
          aria-current={active ? 'page' : undefined}
          aria-label={item.label}
          title={collapsed ? item.label : undefined}
        >
          <Icon size={18} strokeWidth={1.6} aria-hidden='true' />
          <span className={collapsed ? 'kfp-shell-visually-hidden' : 'kfp-shell-nav-label'}>
            {item.label}
          </span>
        </Link>
      </li>
    );
  }

  return (
    <div className='kfp-shell'>
      <a
        href={`#${mainId}`}
        className='kfp-shell-skip-link'
        onClick={(event) => {
          event.preventDefault();
          mainRef.current?.focus();
        }}
      >
        Skip to main content
      </a>
      {!hideSideNav && (
        <aside
          className='kfp-shell-sidebar'
          aria-label='Pipelines sidebar'
          data-collapsed={collapsed}
        >
          <div className='kfp-shell-brand' title={collapsed ? namespace : undefined}>
            <span className='kfp-shell-brand-mark' aria-hidden='true'>
              <KubeflowLogo color='var(--primary-foreground)' style={{ width: 18, height: 18 }} />
            </span>
            <div className={collapsed ? 'kfp-shell-visually-hidden' : 'kfp-shell-brand-copy'}>
              <span className='kfp-shell-brand-title'>Pipelines</span>
              {namespace && <span className='kfp-shell-namespace'>{namespace}</span>}
            </div>
          </div>

          <nav aria-label='Pipeline navigation' className='kfp-shell-navigation'>
            <ul className='kfp-shell-nav-list'>{items.map(renderNavigationItem)}</ul>
            {secondaryItems.length > 0 && (
              <ul className='kfp-shell-nav-list kfp-shell-secondary-nav'>
                {secondaryItems.map(renderNavigationItem)}
              </ul>
            )}
          </nav>

          <footer className='kfp-shell-footer'>
            <label
              htmlFor={themeId}
              className='kfp-shell-theme-control'
              title={collapsed ? `Theme: ${theme}` : undefined}
            >
              <ThemeIcon size={18} strokeWidth={1.6} aria-hidden='true' />
              <span className={collapsed ? 'kfp-shell-visually-hidden' : undefined}>Theme</span>
              <select
                id={themeId}
                aria-label='Theme'
                value={theme}
                onChange={(event) => {
                  const value = event.target.value;
                  if (value === 'system' || value === 'light' || value === 'dark') {
                    setTheme(value);
                  }
                }}
              >
                <option value='system'>System</option>
                <option value='light'>Light</option>
                <option value='dark'>Dark</option>
              </select>
            </label>

            <Button
              variant='ghost'
              className='kfp-shell-collapse'
              disabled={isNarrow}
              aria-label={
                isNarrow
                  ? 'Expand navigation (available on wider screens)'
                  : collapsed
                    ? 'Expand navigation'
                    : 'Collapse navigation'
              }
              aria-expanded={!collapsed}
              title={isNarrow ? 'Navigation is collapsed below 1024 pixels' : undefined}
              onClick={toggleCollapsed}
            >
              {collapsed ? (
                <ChevronRight size={18} strokeWidth={1.6} aria-hidden='true' />
              ) : (
                <ChevronLeft size={18} strokeWidth={1.6} aria-hidden='true' />
              )}
              {!collapsed && <span>Collapse</span>}
            </Button>

            <div className='kfp-shell-footer-links'>
              {footerLinks.map(({ label, href, icon: Icon }) => (
                <a
                  key={label}
                  href={href}
                  aria-label={label}
                  title={label}
                  target='_blank'
                  rel='noopener noreferrer'
                >
                  <Icon size={16} strokeWidth={1.6} aria-hidden='true' />
                  <span className='kfp-shell-visually-hidden'>{label}</span>
                </a>
              ))}
            </div>
            {!collapsed && (version || metadata) && (
              <div
                className='kfp-shell-metadata'
                title={[metadata?.buildDate, metadata?.commitHash].filter(Boolean).join(' · ')}
              >
                {version &&
                  (versionHref ? (
                    <a href={versionHref} target='_blank' rel='noopener noreferrer'>
                      {version}
                    </a>
                  ) : (
                    <span>{version}</span>
                  ))}
                {metadata?.clusterName &&
                  (metadata.clusterHref ? (
                    <a href={metadata.clusterHref} target='_blank' rel='noopener noreferrer'>
                      Cluster: {metadata.clusterName}
                    </a>
                  ) : (
                    <span>Cluster: {metadata.clusterName}</span>
                  ))}
                {metadata?.projectId && <span>Project: {metadata.projectId}</span>}
              </div>
            )}
          </footer>
        </aside>
      )}
      <main id={mainId} ref={mainRef} className='kfp-shell-main' tabIndex={-1}>
        {children}
      </main>
    </div>
  );
}
