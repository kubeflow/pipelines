import React from 'react';
import { Link, LinkProps } from 'react-router-dom';

export const ForwardedLink = React.forwardRef<HTMLAnchorElement, LinkProps>((props, ref) => (
  <Link {...props} ref={ref} />
));
