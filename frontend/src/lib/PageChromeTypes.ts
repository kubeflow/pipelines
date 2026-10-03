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

import type { CSSProperties, ReactNode } from 'react';
import type { NavigateFunction } from 'react-router';
import type { LucideIcon } from 'lucide-react';

export interface ToolbarActionMap {
  [key: string]: ToolbarActionConfig;
}

export interface ToolbarActionConfig {
  action: () => void;
  busy?: boolean;
  disabled?: boolean;
  disabledTitle?: string;
  icon?: LucideIcon;
  id?: string;
  outlined?: boolean;
  primary?: boolean;
  style?: CSSProperties;
  title: string;
  tooltip: string;
}

export interface Breadcrumb {
  displayName: string;
  href: string;
}

export interface ToolbarProps {
  actions: ToolbarActionMap;
  breadcrumbs: Breadcrumb[];
  navigate?: NavigateFunction;
  pageTitle: ReactNode;
  pageTitleTooltip?: string;
  topLevelToolbar?: boolean;
}

export interface BannerProps {
  additionalInfo?: string;
  message?: string;
  mode?: 'error' | 'warning' | 'info';
  showTroubleshootingGuideLink?: boolean;
  refresh?: () => void;
  isLeftAlign?: boolean;
}

export interface SnackbarProps {
  message?: ReactNode;
  open?: boolean;
  autoHideDuration?: number | null;
}
