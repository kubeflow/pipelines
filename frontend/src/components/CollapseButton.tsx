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
import { CompareState } from '../pages/Compare';
import Button from '@material-ui/core/Button';
import ExpandedIcon from '@material-ui/icons/ArrowDropUp';
import { stylesheet, classes } from 'typestyle';
import { color, fontsize } from '../Css';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

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
  collapseSections: { [key: string]: boolean };
  compareSetState: (state: Partial<CompareState>) => void;
  sectionName: string;
  t: TFunction;
}

class CollapseButton extends React.Component<CollapseButtonProps> {
  public render(): JSX.Element {
    const { collapseSections, compareSetState, t } = this.props;
    const sectionName = this.props.sectionName;
    return (
      <div>
        <Button
          onClick={() => {
            collapseSections[sectionName] = !collapseSections[sectionName];
            compareSetState({ collapseSections });
          }}
          title={t('expandCollapseSec')}
          className={css.collapseBtn}
        >
          <ExpandedIcon
            className={classes(css.icon, collapseSections[sectionName] ? css.collapsed : '')}
          />
          {sectionName}
        </Button>
      </div>
    );
  }
}

export default withTranslation('common')(CollapseButton);
