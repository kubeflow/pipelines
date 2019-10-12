import React from 'react';

/** react-linkify decorator that let links default to open in new window. */
export const openLinkInNewWindowDecorator =
  (decoratedHref: string, decoratedText: string, key: number): React.ReactNode =>
    <a href={decoratedHref} key={key} rel='noreferrer noopener' target='_blank'>
      {decoratedText}
    </a>;
