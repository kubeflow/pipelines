/*
 * Copyright 2020 Arrikto Inc.
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

import React, { ReactNode } from 'react';
import { color, fontsize } from '../Css';
import { Card, CardContent, Tooltip, tooltipClasses, TooltipProps } from '@mui/material';
import { styled } from '@mui/material/styles';

const NostyleTooltip = styled(({ className, ...props }: TooltipProps) => (
  <Tooltip {...props} classes={{ popper: className }} />
))({
  [`& .${tooltipClasses.tooltip}`]: {
    backgroundColor: 'transparent',
    border: '0 none',
    color: color.secondaryText,
    fontSize: fontsize.base,
    maxWidth: 220,
  },
});

interface CardTooltipProps {
  helpText?: ReactNode;
  children: React.ReactElement;
}
export const CardTooltip: React.FC<CardTooltipProps> = props => {
  return (
    <NostyleTooltip
      title={
        <Card>
          <CardContent>{props.helpText}</CardContent>
        </Card>
      }
      // v4 used `interactive={true}`; the v5 equivalent is `disableInteractive={false}`.
      disableInteractive={false}
      leaveDelay={400}
      placement='top'
    >
      {props.children}
    </NostyleTooltip>
  );
};
