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

import { useState, type ReactNode } from 'react';
import { Menu } from '@base-ui/react/menu';
import { Check, Filter, X } from 'lucide-react';
import { useTheme } from '../shell/ThemeProvider';
import { Button } from '../ui/button';
import { V2PredicateOperation, type V2Predicate } from '../../apisv2/filter';

export type FilterOption = { value: string; label: string };
export function ColumnFilter({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: readonly FilterOption[];
  onChange: (value: string) => void;
}) {
  const { themeClassName } = useTheme();
  return (
    <Menu.Root>
      <Menu.Trigger
        className='kfp-column-filter'
        aria-label={`Filter ${label.toLowerCase()}`}
        data-active={value !== 'all'}
      >
        <Filter size={13} strokeWidth={2} aria-hidden='true' />
      </Menu.Trigger>
      <Menu.Portal className={themeClassName}>
        <Menu.Positioner align='end' sideOffset={4} className='kfp-menu-positioner'>
          <Menu.Popup className='kfp-menu-menu' aria-label={`Filter ${label.toLowerCase()}`}>
            <Menu.RadioGroup
              value={value}
              onValueChange={(value) => {
                if (typeof value === 'string' && options.some((option) => option.value === value))
                  onChange(value);
              }}
            >
              <div className='kfp-menu-heading'>{label}</div>
              {options.map((option) => (
                <Menu.RadioItem
                  key={option.value}
                  value={option.value}
                  className='kfp-menu-option'
                  closeOnClick
                >
                  {option.label}
                  <Menu.RadioItemIndicator>
                    <Check size={16} aria-hidden='true' />
                  </Menu.RadioItemIndicator>
                </Menu.RadioItem>
              ))}
            </Menu.RadioGroup>
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}

export function FilterChip({ label, onClear }: { label: string; onClear: () => void }) {
  return (
    <Button variant='secondary' size='sm' aria-label={`Clear ${label} filter`} onClick={onClear}>
      {label}
      <X size={13} aria-hidden='true' />
    </Button>
  );
}

export const DATE_OPTIONS = [
  { value: 'all', label: 'Any time' },
  ...['7', '30', '90'].map((value) => ({ value, label: `Last ${value} days` })),
];

/** Keep the selected cutoff stable across refreshes; only a user selection changes it. */
export function DateListFilter({
  label,
  field = 'created_at',
  disabled = false,
  children,
}: {
  label: string;
  field?: string;
  disabled?: boolean;
  children: (filter: {
    predicates: V2Predicate[];
    control: ReactNode;
    chip: ReactNode;
    active: boolean;
  }) => ReactNode;
}) {
  const [selection, setSelection] = useState<{ value: string; since?: Date }>({ value: 'all' });
  const change = (value: string) =>
    setSelection({
      value,
      since: value === 'all' ? undefined : new Date(Date.now() - Number(value) * 86400000),
    });
  const active = !disabled && selection.value !== 'all';
  return children({
    active,
    predicates: active
      ? [
          {
            key: field,
            operation: V2PredicateOperation.GREATER_THAN_EQUALS,
            timestamp_value: selection.since,
          },
        ]
      : [],
    control: disabled ? null : (
      <ColumnFilter
        label={label}
        value={selection.value}
        options={DATE_OPTIONS}
        onChange={change}
      />
    ),
    chip: active ? (
      <FilterChip label={`${label}: last ${selection.value} days`} onClear={() => change('all')} />
    ) : null,
  });
}
