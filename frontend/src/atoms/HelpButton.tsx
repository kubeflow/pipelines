import Card from '@material-ui/core/Card';
import CardContent from '@material-ui/core/CardContent';
import IconButton from '@material-ui/core/IconButton';
import { withStyles } from '@material-ui/core/styles';
import Tooltip from '@material-ui/core/Tooltip';
import HelpIcon from '@material-ui/icons/Help';
import React, { ReactNode } from 'react';
import { color, fontsize } from 'src/Css';

const NostyleTooltip = withStyles({
  tooltip: {
    backgroundColor: 'transparent',
    maxWidth: 220,
    fontSize: fontsize.base,
    color: color.secondaryText,
    border: '0 none',
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
      interactive
      leaveDelay={400}
      placement='top'
    >
      <IconButton>
        <HelpIcon />
      </IconButton>
    </NostyleTooltip>
  );
};
