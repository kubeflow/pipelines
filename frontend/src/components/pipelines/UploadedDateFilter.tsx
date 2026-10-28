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

import { Menu } from '@base-ui/react/menu';
import { Check } from 'lucide-react';
import { ColumnFilter, DATE_OPTIONS } from '../tables/ColumnFilter';
import { V2PredicateOperation, type V2Predicate } from '../../apisv2/filter';

export type UploadedPeriod = 'all' | '7' | '30' | '90';
type Props = { value: UploadedPeriod; onChange: (value: UploadedPeriod) => void };

export function uploadedDatePredicates(since?: Date): V2Predicate[] {
  return since
    ? [
        {
          key: 'created_at',
          operation: V2PredicateOperation.GREATER_THAN_EQUALS,
          timestamp_value: since,
        },
      ]
    : [];
}

export function UploadedDateOptions({ value, onChange }: Props) {
  return (
    <Menu.RadioGroup
      aria-label='Uploaded date'
      value={value}
      onValueChange={(value) => {
        if (value === 'all' || value === '7' || value === '30' || value === '90') onChange(value);
      }}
    >
      <div className='kfp-menu-heading'>Uploaded</div>
      {(['all', '7', '30', '90'] as const).map((period) => (
        <Menu.RadioItem key={period} value={period} className='kfp-menu-option' closeOnClick>
          {period === 'all' ? 'Any time' : `Last ${period} days`}
          <Menu.RadioItemIndicator>
            <Check size={16} aria-hidden='true' />
          </Menu.RadioItemIndicator>
        </Menu.RadioItem>
      ))}
    </Menu.RadioGroup>
  );
}

export function UploadedDateFilter(props: Props) {
  return (
    <ColumnFilter
      label='Uploaded date'
      value={props.value}
      options={DATE_OPTIONS}
      onChange={(value) => {
        if (value === 'all' || value === '7' || value === '30' || value === '90')
          props.onChange(value);
      }}
    />
  );
}
