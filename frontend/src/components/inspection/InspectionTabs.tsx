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

import { Tabs } from '@base-ui/react/tabs';
import { type ReactNode } from 'react';
import './RunInspection.css';

export interface InspectionTabsProps {
  tabs: ReadonlyArray<string | { label: string; disabled?: boolean; tooltip?: string }>;
  selectedTab: number;
  onSwitch: (index: number) => void;
  children: ReactNode;
  ariaLabel?: string;
  className?: string;
}

export function InspectionTabs({
  tabs,
  selectedTab,
  onSwitch,
  children,
  ariaLabel,
  className,
}: InspectionTabsProps) {
  return (
    <Tabs.Root
      className={['kfp-inspection-tabs', className].filter(Boolean).join(' ')}
      value={selectedTab}
      onValueChange={(value: unknown) => {
        if (typeof value === 'number') onSwitch(value);
      }}
    >
      <Tabs.List className='kfp-inspection-tab-list' aria-label={ariaLabel} activateOnFocus={false}>
        {tabs.map((tab, index) => (
          <Tabs.Tab
            key={index}
            value={index}
            disabled={typeof tab === 'string' ? false : tab.disabled}
            title={typeof tab === 'string' ? undefined : tab.tooltip}
            className='kfp-inspection-tab'
          >
            {typeof tab === 'string' ? tab : tab.label}
          </Tabs.Tab>
        ))}
      </Tabs.List>
      <Tabs.Panel className='kfp-inspection-tab-panel' value={selectedTab}>
        {children}
      </Tabs.Panel>
    </Tabs.Root>
  );
}
