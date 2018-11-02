/*
 * Copyright 2018 Google LLC
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

import * as React from 'react';
import Compare from '../pages/Compare';
import Button from '@material-ui/core/Button';
import ExpandedIcon from '@material-ui/icons/ArrowDropUp';
import { stylesheet, classes } from 'typestyle';
import { color, fontsize } from '../Css';

const css = stylesheet({
  collapseBtn: {
    color: color.strong,
    fontSize: fontsize.title,
    fontWeight: 'bold',
    padding: 5,
  },
  collapsed: {
    transform: 'rotate(180deg)',
  },
  icon: {
    marginRight: 5,
    transition: 'transform 0.3s',
  },
});

interface CollapseButtonProps {
  compareComponent: Compare;
  sectionName: string;
}

export default (props: CollapseButtonProps) => {
  const collapseSections = props.compareComponent.state.collapseSections;
  const sectionName = props.sectionName;
  return <div>
    <Button onClick={() => {
      collapseSections[sectionName] = !collapseSections[sectionName];
      props.compareComponent.setState({ collapseSections });
    }} title='Expand/Collapse this section' className={css.collapseBtn}>
      <ExpandedIcon className={classes(
        css.icon,
        collapseSections[sectionName] ? css.collapsed : '',
      )} />
      {sectionName}
    </Button>
  </div>;
};
