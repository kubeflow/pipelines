import React, { DetailedHTMLProps, AnchorHTMLAttributes } from 'react';
import { stylesheet } from 'typestyle';
import { color } from '../Css';
import { classes } from 'typestyle';

const css = stylesheet({
  link: {
    $nest: {
      '&:hover': {
        textDecoration: 'underline',
      },
    },
    color: color.theme,
    textDecoration: 'none',
    wordBreak: 'break-all', // Links do not need to break at words.
  },
});

export const ExternalLink: React.FC<DetailedHTMLProps<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  HTMLAnchorElement
>> = props => (
  // eslint-disable-next-line jsx-a11y/anchor-has-content
  <a {...props} className={classes(css.link, props.className)} target='_blank' rel='noopener' />
);

export const AutoLink: React.FC<DetailedHTMLProps<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  HTMLAnchorElement
>> = props =>
  props.href && props.href.startsWith('#') ? (
    // eslint-disable-next-line jsx-a11y/anchor-has-content
    <a {...props} className={classes(css.link, props.className)} />
  ) : (
    <ExternalLink {...props} />
  );
