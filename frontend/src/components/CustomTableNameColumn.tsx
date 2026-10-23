/**
 * Copyright 2021 The Kubeflow Authors
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

import { CustomRendererProps } from './CustomTable';
import React, { type ReactElement, useId, useState } from 'react';
import { Tooltip } from '@base-ui/react/tooltip';

export function NameTooltip({ name, children }: { name: string; children: ReactElement }) {
  const [trigger, setTrigger] = useState<HTMLElement | null>(null);
  const [open, setOpen] = useState(false);
  const triggerId = useId();
  return (
    <Tooltip.Root open={open} onOpenChange={setOpen} triggerId={triggerId}>
      <Tooltip.Trigger
        id={triggerId}
        ref={setTrigger}
        render={children}
        closeOnClick={false}
        onPointerDown={(event) => {
          if (event.pointerType === 'touch') setOpen(true);
        }}
      />
      <Tooltip.Portal container={trigger?.closest<HTMLElement>('.kfp-theme') ?? undefined}>
        <Tooltip.Positioner side='top' sideOffset={8} className='kfp-shell-utility-positioner'>
          <Tooltip.Popup role='tooltip' className='kfp-shell-utility-tooltip'>
            {name}
          </Tooltip.Popup>
        </Tooltip.Positioner>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
}

/**
 * Common name custom renderer that shows a tooltip when hovered. The tooltip helps if there isn't
 * enough space to show the entire name in limited space.
 */
export const NameWithTooltip: React.FC<
  CustomRendererProps<{
    display_name?: string;
    name?: string;
  }>
> = (props: CustomRendererProps<{ display_name?: string; name?: string }>) => {
  return (
    <NameTooltip name={'Name: ' + (props.value?.name || '')}>
      <span tabIndex={0} aria-label={'Name: ' + (props.value?.name || '')}>
        {props.value?.display_name || props.value?.name || ''}
      </span>
    </NameTooltip>
  );
};
