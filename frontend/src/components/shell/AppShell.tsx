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
import type { ReactElement, ReactNode } from 'react';
import { Link } from 'react-router';
import { Menu } from '@base-ui/react/menu';
import { Tooltip } from '@base-ui/react/tooltip';
import {
  BookOpen,
  Check,
  PanelLeftClose,
  PanelLeftOpen,
  CodeXml,
  MessageSquare,
  Monitor,
  Moon,
  Search,
  Sun,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import KubeflowLogo from 'src/icons/kubeflowLogo';
import { LocalStorage } from 'src/lib/LocalStorage';
import { Button } from '../ui/button';
import { useTheme } from './ThemeProvider';
import './AppShell.css';

export interface AppShellNavItem {
  id: string;
  label: string;
  href: string;
  icon: LucideIcon;
  active?: boolean;
  elementId?: string;
}

export interface AppShellProps {
  children: ReactNode;
  items: readonly AppShellNavItem[];
  currentPath: string;
  secondaryItems?: readonly AppShellNavItem[];
  hideSideNav?: boolean;
  namespace?: string;
  onSearch?: () => void;
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

function FooterTooltip({ label, children }: { label: string; children: ReactElement }) {
  const { themeClassName } = useTheme();
  const tooltipId = useId();
  return (
    <Tooltip.Root>
      <Tooltip.Trigger render={children} aria-describedby={tooltipId} />
      <Tooltip.Portal className={themeClassName}>
        <Tooltip.Positioner side='top' sideOffset={8} className='kfp-shell-utility-positioner'>
          <Tooltip.Popup id={tooltipId} role='tooltip' className='kfp-shell-utility-tooltip'>
            {label}
          </Tooltip.Popup>
        </Tooltip.Positioner>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
}

export function AppShell({
  children,
  items,
  currentPath,
  secondaryItems = [],
  hideSideNav = false,
  namespace,
  onSearch,
  version,
  versionHref,
  metadata,
}: AppShellProps) {
  const [preferredCollapsed, setPreferredCollapsed] = useState(LocalStorage.isNavbarCollapsed);
  const isNarrow = useSyncExternalStore(subscribeToViewport, isNarrowViewport, () => false);
  const collapsed = isNarrow || preferredCollapsed;
  const { theme, themeClassName, setTheme } = useTheme();
  const mainId = useId();
  const appearanceId = useId();
  const mainRef = useRef<HTMLElement>(null);
  const themeLabel = `Theme: ${theme[0].toUpperCase()}${theme.slice(1)}`;
  const collapseLabel = isNarrow
    ? 'Expand navigation (available on wider screens)'
    : collapsed
      ? 'Expand navigation'
      : 'Collapse navigation';
  const ThemeIcon = theme === 'system' ? Monitor : theme === 'dark' ? Moon : Sun;

  function toggleCollapsed() {
    const next = !preferredCollapsed;
    setPreferredCollapsed(next);
    LocalStorage.saveNavbarCollapsed(next);
  }

  function renderNavigationItem(item: AppShellNavItem) {
    const Icon = item.icon;
    const active = item.active ?? currentPath === item.href;
    return (
      <li key={item.id}>
        <Link
          id={item.elementId}
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

          {onSearch && (
            <div className='kfp-shell-search-container'>
              <Button
                variant='ghost'
                className='kfp-shell-search'
                aria-label='Search'
                aria-keyshortcuts='Control+k Meta+k'
                title='Search (Ctrl/Cmd+K)'
                onClick={onSearch}
              >
                <Search size={18} strokeWidth={1.6} aria-hidden='true' />
                {!collapsed && (
                  <>
                    <span>Search</span>
                    <kbd aria-hidden='true'>Ctrl / ⌘ K</kbd>
                  </>
                )}
              </Button>
            </div>
          )}
          <nav aria-label='Pipeline navigation' className='kfp-shell-navigation'>
            <ul className='kfp-shell-nav-list'>{items.map(renderNavigationItem)}</ul>
            {secondaryItems.length > 0 && (
              <ul className='kfp-shell-nav-list kfp-shell-secondary-nav'>
                {secondaryItems.map(renderNavigationItem)}
              </ul>
            )}
          </nav>

          <footer className='kfp-shell-footer'>
            <Tooltip.Provider>
              <div className='kfp-shell-footer-tools'>
                <Menu.Root>
                  <FooterTooltip label={themeLabel}>
                    <Menu.Trigger
                      render={
                        <Button
                          variant='ghost'
                          size='icon'
                          className='kfp-shell-utility'
                          aria-label={themeLabel}
                        />
                      }
                    >
                      <ThemeIcon size={18} strokeWidth={1.6} aria-hidden='true' />
                    </Menu.Trigger>
                  </FooterTooltip>
                  <Menu.Portal className={themeClassName}>
                    <Menu.Positioner
                      side={collapsed ? 'right' : 'top'}
                      align='start'
                      sideOffset={8}
                      className='kfp-shell-utility-positioner'
                    >
                      <Menu.Popup className='kfp-shell-theme-menu' aria-labelledby={appearanceId}>
                        <div id={appearanceId} className='kfp-shell-theme-heading'>
                          Appearance
                        </div>
                        <Menu.RadioGroup
                          value={theme}
                          onValueChange={(value: unknown) => {
                            if (value === 'light' || value === 'dark' || value === 'system')
                              setTheme(value);
                          }}
                        >
                          {(
                            [
                              { value: 'light', label: 'Light', icon: Sun },
                              { value: 'dark', label: 'Dark', icon: Moon },
                              { value: 'system', label: 'System', icon: Monitor },
                            ] as const
                          ).map(({ value, label, icon: Icon }) => (
                            <Menu.RadioItem
                              key={value}
                              value={value}
                              className='kfp-shell-theme-option'
                              closeOnClick
                            >
                              <Icon size={18} strokeWidth={1.6} aria-hidden='true' />
                              {label}
                              <Menu.RadioItemIndicator className='kfp-shell-theme-check'>
                                <Check size={16} aria-hidden='true' />
                              </Menu.RadioItemIndicator>
                            </Menu.RadioItem>
                          ))}
                        </Menu.RadioGroup>
                      </Menu.Popup>
                    </Menu.Positioner>
                  </Menu.Portal>
                </Menu.Root>
                {footerLinks.map(({ label, href, icon: Icon }) => (
                  <FooterTooltip key={label} label={label}>
                    <a
                      className='kfp-shell-utility'
                      href={href}
                      aria-label={label}
                      target='_blank'
                      rel='noopener noreferrer'
                    >
                      <Icon size={18} strokeWidth={1.6} aria-hidden='true' />
                    </a>
                  </FooterTooltip>
                ))}
                <FooterTooltip label={collapseLabel}>
                  <Button
                    variant='ghost'
                    size='icon'
                    className='kfp-shell-utility kfp-shell-collapse'
                    disabled={isNarrow}
                    aria-label={collapseLabel}
                    aria-expanded={!collapsed}
                    onClick={toggleCollapsed}
                  >
                    {collapsed ? (
                      <PanelLeftOpen size={18} strokeWidth={1.6} aria-hidden='true' />
                    ) : (
                      <PanelLeftClose size={18} strokeWidth={1.6} aria-hidden='true' />
                    )}
                  </Button>
                </FooterTooltip>
              </div>
            </Tooltip.Provider>
            {!collapsed && (
              <div
                className='kfp-shell-metadata'
                title={[metadata?.buildDate, metadata?.commitHash].filter(Boolean).join(' · ')}
              >
                {version &&
                  (versionHref ? (
                    <a
                      className='kfp-shell-metadata-version'
                      href={versionHref}
                      target='_blank'
                      rel='noopener noreferrer'
                    >
                      {version}
                    </a>
                  ) : (
                    <span className='kfp-shell-metadata-version'>{version}</span>
                  ))}
                {metadata?.clusterName &&
                  (metadata.clusterHref ? (
                    <a
                      className='kfp-shell-metadata-cluster'
                      href={metadata.clusterHref}
                      target='_blank'
                      rel='noopener noreferrer'
                    >
                      Cluster: {metadata.clusterName}
                    </a>
                  ) : (
                    <span className='kfp-shell-metadata-cluster'>
                      Cluster: {metadata.clusterName}
                    </span>
                  ))}
                {metadata?.projectId && (
                  <span className='kfp-shell-metadata-project'>Project: {metadata.projectId}</span>
                )}
              </div>
            )}
          </footer>
        </aside>
      )}
      <main id={mainId} ref={mainRef} className='kfp-shell-main' tabIndex={-1}>
        {hideSideNav && onSearch && (
          <div className='kfp-shell-embedded-search'>
            <Button
              variant='ghost'
              size='sm'
              aria-label='Search'
              aria-keyshortcuts='Control+k Meta+k'
              onClick={onSearch}
            >
              <Search size={16} aria-hidden='true' /> Search
            </Button>
          </div>
        )}
        {children}
      </main>
    </div>
  );
}
