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

import { Link } from 'react-router';
import { Menu } from '@base-ui/react/menu';
import { ChevronDown, Ellipsis } from 'lucide-react';
import { RoutePage } from '../Router';
import { Button } from '../ui/button';
import { useTheme } from './ThemeProvider';

export function WorkspaceMenu({ footer = false }: { footer?: boolean }) {
  const { themeClassName } = useTheme();
  return (
    <Menu.Root>
      <Menu.Trigger
        render={
          <Button
            variant='ghost'
            size={footer ? 'icon' : 'sm'}
            className={footer ? 'kfp-shell-utility' : undefined}
            aria-label={footer ? 'More workspace actions' : 'Workspace'}
            title={footer ? 'More workspace actions' : undefined}
          />
        }
      >
        {footer ? (
          <Ellipsis size={18} aria-hidden='true' />
        ) : (
          <>
            <ChevronDown size={16} aria-hidden='true' />
            Workspace
          </>
        )}
      </Menu.Trigger>
      <Menu.Portal className={themeClassName}>
        <Menu.Positioner
          side={footer ? 'top' : 'bottom'}
          align='end'
          sideOffset={4}
          className='kfp-menu-positioner'
        >
          <Menu.Popup className='kfp-menu-menu' aria-label='Workspace'>
            <Menu.Item
              render={<Link to={RoutePage.METADATA_TRANSFER} />}
              className='kfp-menu-option'
            >
              Import / export namespace data…
            </Menu.Item>
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}
