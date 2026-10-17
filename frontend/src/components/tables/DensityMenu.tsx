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
import { Check, SlidersHorizontal } from 'lucide-react';
import { Button } from '../ui/button';
import { useTheme } from '../shell/ThemeProvider';
import { useDensity } from './DensityProvider';

export function DensityMenu() {
  const preference = useDensity();
  return preference ? <DensityMenuContent preference={preference} /> : null;
}

function DensityMenuContent({
  preference,
}: {
  preference: NonNullable<ReturnType<typeof useDensity>>;
}) {
  const { themeClassName } = useTheme();
  return (
    <Menu.Root>
      <Menu.Trigger render={<Button variant='secondary' size='sm' aria-label='Display options' />}>
        <SlidersHorizontal size={16} aria-hidden='true' />
        Display
      </Menu.Trigger>
      <Menu.Portal className={themeClassName}>
        <Menu.Positioner align='end' sideOffset={4} className='kfp-density-positioner'>
          <Menu.Popup className='kfp-density-menu' aria-label='Display options'>
            <Menu.RadioGroup
              aria-label='Collection density'
              value={preference.density}
              onValueChange={(value: unknown) => {
                if (value === 'comfortable' || value === 'compact') preference.setDensity(value);
              }}
            >
              <div className='kfp-density-heading'>Density · all lists</div>
              {(['comfortable', 'compact'] as const).map((value) => (
                <Menu.RadioItem
                  key={value}
                  value={value}
                  className='kfp-density-option'
                  closeOnClick
                >
                  {value === 'comfortable' ? 'Comfortable' : 'Compact'}
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
