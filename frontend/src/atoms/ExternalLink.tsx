import React, { DetailedHTMLProps, AnchorHTMLAttributes } from 'react';
import { stylesheet } from 'typestyle';
import { color } from '../Css';

const css = stylesheet({
  link: {
    $nest: {
      '&:hover': {
        textDecoration: 'underline',
      },
    },
    color: color.theme,
    textDecoration: 'none',
  },
});

export const ExternalLink: React.FC<
  DetailedHTMLProps<AnchorHTMLAttributes<HTMLAnchorElement>, HTMLAnchorElement>
> = props => <a {...props} className={css.link} target='_blank' rel='noreferrer noopener' />;
