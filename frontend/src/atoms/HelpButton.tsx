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

import HelpIcon from '@mui/icons-material/Help';
import React, { ReactNode } from 'react';
import { CardTooltip } from './CardTooltip';
import { IconButton } from '@mui/material';

interface HelpButtonProps {
  helpText?: ReactNode;
}
export const HelpButton: React.FC<HelpButtonProps> = ({ helpText }) => {
  return (
    <CardTooltip helpText={helpText}>
      <IconButton size='large'>
        <HelpIcon />
      </IconButton>
    </CardTooltip>
  );
};
