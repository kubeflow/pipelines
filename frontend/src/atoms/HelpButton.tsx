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
