import IconButton from '@material-ui/core/IconButton';
import HelpIcon from '@material-ui/icons/Help';
import React, { ReactNode } from 'react';
import { CardTooltip } from './CardTooltip';

interface HelpButtonProps {
  helpText?: ReactNode;
}
export const HelpButton: React.FC<HelpButtonProps> = ({ helpText }) => {
  return (
    <CardTooltip helpText={helpText}>
      <IconButton>
        <HelpIcon />
      </IconButton>
    </CardTooltip>
  );
};
