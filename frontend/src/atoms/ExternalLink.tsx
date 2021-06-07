/**
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
