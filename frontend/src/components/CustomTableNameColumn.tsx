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
