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
import React from 'react';
import Tooltip from '@material-ui/core/Tooltip';

/**
 * Common name custom renderer that shows a tooltip when hovered. The tooltip helps if there isn't
 * enough space to show the entire name in limited space.
 */
export const NameWithTooltip: React.FC<CustomRendererProps<string>> = (
  props: CustomRendererProps<string>,
) => {
  return (
    <Tooltip title={props.value || ''} enterDelay={300} placement='top-start'>
      <span>{props.value || ''}</span>
    </Tooltip>
  );
};
