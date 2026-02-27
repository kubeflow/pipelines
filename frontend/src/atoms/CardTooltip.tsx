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
      leaveDelay={400}
      placement='top'
    >
      {props.children}
    </NostyleTooltip>
  );
};
