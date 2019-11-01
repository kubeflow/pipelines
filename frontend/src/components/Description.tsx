/*
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import Markdown from 'markdown-to-jsx';
import { stylesheet } from 'typestyle';
import { color } from '../Css';

function preventEventBubbling(e: React.MouseEvent): void {
  e.stopPropagation();
}

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

const renderExternalLink = (props: {}) => (
  <a
    {...props}
    className={css.link}
    target='_blank'
    rel='noreferrer noopener'
    onClick={preventEventBubbling}
  />
);

const options = {
  overrides: { a: { component: renderExternalLink } },
};

const optionsForceInline = {
  ...options,
  forceInline: true,
};

export const Description: React.FC<{ description: string; forceInline?: boolean }> = ({
  description,
  forceInline,
}) => {
  return <Markdown options={forceInline ? optionsForceInline : options}>{description}</Markdown>;
};
