import Card from '@material-ui/core/Card';
import CardContent from '@material-ui/core/CardContent';
import IconButton from '@material-ui/core/IconButton';
import { withStyles } from '@material-ui/core/styles';
import Tooltip from '@material-ui/core/Tooltip';
import HelpIcon from '@material-ui/icons/Help';
import React, { ReactNode } from 'react';
import { color, fontsize } from '../Css';

const NostyleTooltip = withStyles({
  tooltip: {
    backgroundColor: 'transparent',
    border: '0 none',
    color: color.secondaryText,
    fontSize: fontsize.base,
    maxWidth: 220,
  },
})(Tooltip);

interface HelpButtonProps {
  helpText?: ReactNode;
}
export const HelpButton: React.FC<HelpButtonProps> = ({ helpText }) => {
  return (
    <NostyleTooltip
      title={
        <Card>
          <CardContent>{helpText}</CardContent>
        </Card>
      }
      interactive={true}
      leaveDelay={400}
      placement='top'
    >
      <IconButton>
        <HelpIcon />
      </IconButton>
    </NostyleTooltip>
  );
};
