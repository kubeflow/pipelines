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

import { useEffect, useEffectEvent, useId, useState, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { Dialog } from '@base-ui/react/dialog';
import {
  Archive,
  ArrowLeft,
  ArrowUpFromLine,
  ChevronRight,
  Copy,
  GitCompareArrows,
  LoaderCircle,
  Plus,
  RefreshCw,
  Trash2,
  X,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { SnackbarProps } from '@mui/material/Snackbar';
import type { ToolbarProps } from '../Toolbar';
import type { BannerProps } from '../Banner';
import type { DialogProps } from '../Router';
import { Button } from '../ui/button';
import { useTheme } from './ThemeProvider';
import './ModernPageChrome.css';

export interface ModernPageChromeProps {
  children: ReactNode;
  toolbarProps: ToolbarProps;
  bannerProps: BannerProps;
  dialogProps: DialogProps;
  snackbarProps: SnackbarProps;
  onDialogClose: (action?: () => void) => void;
  onSnackbarClose: () => void;
  navigationNotice?: ReactNode;
  showThemeControl?: boolean;
}

const actionIcons: Record<string, LucideIcon> = {
  archive: Archive,
  cloneRun: Copy,
  compare: GitCompareArrows,
  deleteRun: Trash2,
  newRun: Plus,
  refresh: RefreshCw,
  restore: ArrowUpFromLine,
};

function ThemeControl() {
  const { theme, setTheme } = useTheme();
  const id = useId();
  return (
    <label className='kfp-page-theme' htmlFor={id}>
      Theme
      <select
        id={id}
        value={theme}
        onChange={(event) => {
          const value = event.target.value;
          if (value === 'system' || value === 'light' || value === 'dark') setTheme(value);
        }}
      >
        <option value='system'>System</option>
        <option value='light'>Light</option>
        <option value='dark'>Dark</option>
      </select>
    </label>
  );
}

function PageToolbar({
  toolbar,
  showThemeControl,
}: {
  toolbar: ToolbarProps;
  showThemeControl?: boolean;
}) {
  return (
    <header className='kfp-page-header'>
      <div className='kfp-page-heading'>
        {toolbar.breadcrumbs.length > 0 && (
          <nav aria-label='Breadcrumbs' className='kfp-page-breadcrumbs'>
            {toolbar.breadcrumbs.map((crumb, index) => (
              <span key={`${crumb.href}:${index}`}>
                {index > 0 && <ChevronRight size={12} aria-hidden='true' />}
                <Link to={crumb.href}>{crumb.displayName}</Link>
              </span>
            ))}
          </nav>
        )}
        <div className='kfp-page-title-row'>
          {toolbar.breadcrumbs.length > 0 && (
            <Button
              variant='ghost'
              size='icon'
              aria-label='Back'
              disabled={!toolbar.navigate || window.history.length < 2}
              onClick={() => toolbar.navigate?.(-1)}
            >
              <ArrowLeft size={18} aria-hidden='true' />
            </Button>
          )}
          <h1 data-testid='page-title' title={toolbar.pageTitleTooltip}>
            {toolbar.pageTitle}
          </h1>
        </div>
      </div>
      <div className='kfp-page-actions'>
        {showThemeControl && <ThemeControl />}
        {Object.entries(toolbar.actions).map(([key, action]) => {
          const Icon = actionIcons[key] || action.icon;
          const explanation =
            action.disabled && action.disabledTitle ? action.disabledTitle : action.tooltip;
          return (
            <span key={key} title={explanation} style={action.style}>
              <Button
                id={action.id}
                variant={action.primary ? 'default' : 'secondary'}
                disabled={action.disabled || action.busy}
                aria-busy={action.busy || undefined}
                aria-description={explanation}
                onClick={action.action}
              >
                {action.busy ? (
                  <LoaderCircle size={16} className='kfp-page-spinner' aria-hidden='true' />
                ) : (
                  Icon && <Icon size={16} aria-hidden='true' />
                )}
                {action.title}
              </Button>
            </span>
          );
        })}
      </div>
    </header>
  );
}

function PageDialog({
  open,
  title,
  children,
  actions,
  onClose,
}: {
  open: boolean;
  title: ReactNode;
  children?: ReactNode;
  actions: ReactNode;
  onClose: () => void;
}) {
  const { resolvedTheme } = useTheme();
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onClose();
      }}
    >
      {/* Portals are outside the application theme element, so inherit the resolved palette explicitly. */}
      <Dialog.Portal className={`kfp-theme ${resolvedTheme === 'dark' ? 'dark' : ''}`}>
        <Dialog.Backdrop className='kfp-page-dialog-backdrop' />
        <Dialog.Viewport className='kfp-page-dialog-viewport'>
          <Dialog.Popup className='kfp-page-dialog'>
            <Dialog.Title className='kfp-page-dialog-title'>{title}</Dialog.Title>
            {children && (
              <Dialog.Description className='kfp-page-dialog-content'>
                {children}
              </Dialog.Description>
            )}
            <div className='kfp-page-dialog-actions'>{actions}</div>
          </Dialog.Popup>
        </Dialog.Viewport>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function PageBanner({ banner }: { banner: BannerProps }) {
  const [detailsOpen, setDetailsOpen] = useState(false);
  const mode = banner.mode || 'error';
  return (
    <>
      <div className='kfp-page-banner' data-mode={mode} role={mode === 'info' ? 'status' : 'alert'}>
        <p>{banner.message}</p>
        <div className='kfp-page-banner-actions'>
          {mode === 'error' && (
            <a href='https://www.kubeflow.org/docs/pipelines/troubleshooting'>
              Troubleshooting guide
            </a>
          )}
          {banner.additionalInfo && (
            <Button variant='secondary' size='sm' onClick={() => setDetailsOpen(true)}>
              Details
            </Button>
          )}
          {mode !== 'info' && banner.refresh && (
            <Button variant='secondary' size='sm' onClick={banner.refresh}>
              Refresh
            </Button>
          )}
        </div>
      </div>
      <PageDialog
        open={detailsOpen}
        title={mode === 'error' ? 'An error occurred' : mode === 'warning' ? 'Warning' : 'Info'}
        onClose={() => setDetailsOpen(false)}
        actions={<Button onClick={() => setDetailsOpen(false)}>Dismiss</Button>}
      >
        {banner.additionalInfo}
      </PageDialog>
    </>
  );
}

function subscribeToWindowActivity(onChange: () => void) {
  window.addEventListener('focus', onChange);
  window.addEventListener('blur', onChange);
  document.addEventListener('visibilitychange', onChange);
  return () => {
    window.removeEventListener('focus', onChange);
    window.removeEventListener('blur', onChange);
    document.removeEventListener('visibilitychange', onChange);
  };
}

function isWindowInactive() {
  return document.visibilityState === 'hidden' || !document.hasFocus();
}

function PageNotification({
  message,
  duration,
  onClose,
}: {
  message: ReactNode;
  duration?: number | null;
  onClose: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const inactiveWindow = useSyncExternalStore(
    subscribeToWindowActivity,
    isWindowInactive,
    () => false,
  );
  const paused = hovered || focused || inactiveWindow;
  const close = useEffectEvent(onClose);
  // External sync: schedule the existing auto-dismiss contract; focus/hover pauses reading time.
  useEffect(() => {
    if (paused || duration == null) return;
    const timer = window.setTimeout(() => close(), duration);
    return () => window.clearTimeout(timer);
  }, [duration, paused]);
  return (
    <div
      className='kfp-page-notification'
      role='status'
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onFocusCapture={() => setFocused(true)}
      onBlurCapture={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setFocused(false);
      }}
    >
      <span>{message}</span>
      <Button variant='ghost' size='icon' aria-label='Dismiss notification' onClick={onClose}>
        <X size={16} aria-hidden='true' />
      </Button>
    </div>
  );
}

export function ModernPageChrome({
  children,
  toolbarProps,
  bannerProps,
  dialogProps,
  snackbarProps,
  onDialogClose,
  onSnackbarClose,
  navigationNotice,
  showThemeControl,
}: ModernPageChromeProps) {
  return (
    <section className='kfp-modern-page' aria-label='Runs workspace'>
      <PageToolbar toolbar={toolbarProps} showThemeControl={showThemeControl} />
      {navigationNotice}
      {bannerProps.message && (
        <PageBanner
          key={`${bannerProps.mode}:${bannerProps.message}:${bannerProps.additionalInfo}`}
          banner={bannerProps}
        />
      )}
      <div className='kfp-modern-page-content'>{children}</div>
      <PageDialog
        open={dialogProps.open !== false}
        title={dialogProps.title || 'Confirm action'}
        onClose={() => onDialogClose()}
        actions={dialogProps.buttons?.map((button, index) => (
          <Button
            key={index}
            variant={button.text === 'Cancel' ? 'secondary' : 'default'}
            onClick={() => onDialogClose(button.onClick)}
          >
            {button.text}
          </Button>
        ))}
      >
        {dialogProps.content}
      </PageDialog>
      {snackbarProps.open && (
        <PageNotification
          key={typeof snackbarProps.message === 'string' ? snackbarProps.message : undefined}
          message={snackbarProps.message}
          duration={snackbarProps.autoHideDuration}
          onClose={onSnackbarClose}
        />
      )}
    </section>
  );
}
