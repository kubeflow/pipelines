import React, { DetailedHTMLProps, AnchorHTMLAttributes } from 'react';
import { stylesheet } from 'typestyle';
import { color } from '../Css';

export const linkStyles = stylesheet({
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

export const ExternalLink: React.FC<DetailedHTMLProps<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  HTMLAnchorElement
>> = props => (
  // eslint-disable-next-line jsx-a11y/anchor-has-content
  <a {...props} className={linkStyles.link} target='_blank' rel='noopener' />
);

export const AutoLink: React.FC<DetailedHTMLProps<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  HTMLAnchorElement
>> = props =>
  props.href && props.href.startsWith('#') ? (
    // eslint-disable-next-line jsx-a11y/anchor-has-content
    <a {...props} className={linkStyles.link} />
  ) : (
    <ExternalLink {...props} />
  );
