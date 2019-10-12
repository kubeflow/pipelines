import React from 'react';
import { stylesheet } from 'typestyle';

const css = stylesheet({
  link: {
    // Links are usually long. Use 'break-all' to avoid unbalanced formatting
    // with long blank space before and after a long link.
    wordBreak: 'break-all',
  }
});

/** react-linkify decorator that let links default to open in new window. */
export const openLinkInNewWindowDecorator =
  (decoratedHref: string, decoratedText: string, key: number): React.ReactNode =>
    <a
      href={decoratedHref}
      key={key}
      rel='noreferrer noopener'
      target='_blank'
      className={css.link}
    >
      {decoratedText}
    </a>;
