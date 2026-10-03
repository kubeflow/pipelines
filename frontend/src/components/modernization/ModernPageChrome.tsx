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

import type { SnackbarProps, ToolbarProps, BannerProps } from 'src/lib/PageChromeTypes';
import { useEffect, useEffectEvent, useId, useState, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import {
  Archive,
  ArrowLeft,
  ArrowUpFromLine,
  ChevronRight,
  ChevronsDownUp,
  ChevronsUpDown,
  Copy,
  GitCompareArrows,
  LoaderCircle,
  Plus,
  Pause,
  Play,
  Repeat,
  RefreshCw,
  RotateCcw,
  Square,
  Trash2,
  Upload,
  X,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { DialogProps } from '../Router';
import { Button } from '../ui/button';
import { ModalDialog } from '../ui/dialog';
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
  reserveBreadcrumbHeader?: boolean;
}

const actionIcons: Record<string, LucideIcon> = {
  archive: Archive,
  cloneRun: Copy,
  cloneRecurringRun: Copy,
  collapse: ChevronsDownUp,
  expand: ChevronsUpDown,
  disableRecurringRun: Pause,
  enableRecurringRun: Play,
  newExperiment: Plus,
  newPipelineVersion: Upload,
  newRecurringRun: Repeat,
  newRunFromPipelineVersion: Plus,
  uploadPipeline: Upload,
  compare: GitCompareArrows,
  deleteRun: Trash2,
  newRun: Plus,
  refresh: RefreshCw,
  restore: ArrowUpFromLine,
  retry: RotateCcw,
  terminateRun: Square,
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
  reserveBreadcrumbHeader = false,
}: {
  toolbar: ToolbarProps;
  showThemeControl?: boolean;
  reserveBreadcrumbHeader?: boolean;
}) {
  const empty =
    !Object.keys(toolbar.actions).length &&
    !toolbar.breadcrumbs.length &&
    !toolbar.pageTitle &&
    !showThemeControl;
  const reserveSpace = reserveBreadcrumbHeader && toolbar.topLevelToolbar !== false;
  if (empty && !reserveSpace) return null;
  const Heading = toolbar.topLevelToolbar === false ? 'h2' : 'h1';
  return (
    <header
      className='kfp-page-header'
      data-embedded={toolbar.topLevelToolbar === false}
      data-reserve-breadcrumb-header={reserveSpace || undefined}
      aria-hidden={empty || undefined}
    >
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
          {toolbar.pageTitle && (
            <Heading data-testid='page-title' title={toolbar.pageTitleTooltip}>
              {toolbar.pageTitle}
            </Heading>
          )}
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

export function ModernToolbar(props: ToolbarProps) {
  if (!Object.keys(props.actions).length && !props.breadcrumbs.length && !props.pageTitle) {
    return null;
  }
  return <PageToolbar toolbar={props} />;
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
      <ModalDialog
        open={detailsOpen}
        title={mode === 'error' ? 'An error occurred' : mode === 'warning' ? 'Warning' : 'Info'}
        onClose={() => setDetailsOpen(false)}
        actions={<Button onClick={() => setDetailsOpen(false)}>Dismiss</Button>}
      >
        {banner.additionalInfo}
      </ModalDialog>
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
  reserveBreadcrumbHeader,
}: ModernPageChromeProps) {
  return (
    <section className='kfp-modern-page' aria-label='Pipeline workspace'>
      <PageToolbar
        toolbar={toolbarProps}
        showThemeControl={showThemeControl}
        reserveBreadcrumbHeader={reserveBreadcrumbHeader}
      />
      {navigationNotice}
      {bannerProps.message && (
        <PageBanner
          key={`${bannerProps.mode}:${bannerProps.message}:${bannerProps.additionalInfo}`}
          banner={bannerProps}
        />
      )}
      <div className='kfp-modern-page-content'>{children}</div>
      <ModalDialog
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
      </ModalDialog>
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
