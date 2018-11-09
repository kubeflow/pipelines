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
import ArrowBackIcon from '@material-ui/icons/ArrowBack';
import BusyButton from '../atoms/BusyButton';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';
import IconButton from '@material-ui/core/IconButton';
import Tooltip from '@material-ui/core/Tooltip';
import { History } from 'history';
import { Link } from 'react-router-dom';
import { classes, stylesheet } from 'typestyle';
import { spacing, fonts, fontsize, color, dimension, commonCss } from '../Css';

export interface ToolbarActionConfig {
  action: () => void;
  busy?: boolean;
  disabled?: boolean;
  disabledTitle?: string;
  icon?: any;
  id?: string;
  outlined?: boolean;
  title: string;
  tooltip: string;
}

export interface Breadcrumb {
  displayName: string;
  href: string;
}

const backIconHeight = 24;

const css = stylesheet({
  actions: {
    display: 'flex',
    marginRight: spacing.units(-2),
  },
  backIcon: {
    color: color.foreground,
    fontSize: backIconHeight,
    verticalAlign: 'bottom',
  },
  backLink: {
    cursor: 'pointer',
    marginRight: 10,
    padding: 3,
  },
  breadcrumbs: {
    color: color.inactive,
    fontFamily: fonts.secondary,
    fontSize: fontsize.small,
    letterSpacing: 0.25,
    margin: '10px 37px',
  },
  chevron: {
    height: 12,
  },
  link: {
    $nest: {
      '&:hover': {
        background: color.hoverBg,
      }
    },
    borderRadius: 3,
    padding: 3,
  },
  pageName: {
    color: color.strong,
    fontSize: fontsize.title,
    lineHeight: `${backIconHeight}px`,
  },
  root: {
    alignItems: 'center',
    display: 'flex',
    flexShrink: 0,
    height: dimension.jumbo,
    justifyContent: 'space-between',
  },
  topLevelToolbar: {
    borderBottom: '1px solid #eee',
    paddingBottom: 15,
    paddingLeft: 20,
  },
});

export interface ToolbarProps {
  actions: ToolbarActionConfig[];
  breadcrumbs: Breadcrumb[];
  history?: History;
  topLevelToolbar?: boolean;
}

class Toolbar extends React.Component<ToolbarProps> {

  public render(): JSX.Element {
    const currentPage = this.props.breadcrumbs.length ?
      this.props.breadcrumbs[this.props.breadcrumbs.length - 1].displayName : '';
    const breadcrumbs = this.props.breadcrumbs.slice(0, this.props.breadcrumbs.length - 1);

    return (
      <div className={
        classes(css.root, (this.props.topLevelToolbar !== false) && css.topLevelToolbar)}>
        <div style={{ minWidth: 100 }}>
          {/* Breadcrumb */}
          <div className={classes(css.breadcrumbs, commonCss.flex)}>
            {breadcrumbs.map((crumb, i) => (
              <span className={commonCss.flex} key={i} title={crumb.displayName}>
                {i !== 0 && <ChevronRightIcon className={css.chevron} />}
                <Link className={classes(commonCss.unstyled, commonCss.ellipsis, css.link)}
                  to={crumb.href}>
                  {crumb.displayName}
                </Link>
              </span>
            ))}
          </div>
          <div className={commonCss.flex}>
            {/* Back Arrow */}
            {breadcrumbs.length > 0 &&
              <Tooltip title={'Back'} enterDelay={300}>
                <IconButton className={css.backLink}
                  // Need to handle this for when browsing back doesn't make sense
                  onClick={this.props.history!.goBack}>
                  <ArrowBackIcon className={css.backIcon} />
                </IconButton>
              </Tooltip>}
            {/* Resource Name */}
            <span className={classes(css.pageName, commonCss.ellipsis)} title={currentPage}>
              {currentPage}
            </span>
          </div>
        </div>
        {/* Actions / Buttons */}
        <div className={css.actions}>
          {this.props.actions.map((b, i) => (
            <Tooltip title={(b.disabled && b.disabledTitle) ? b.disabledTitle : b.tooltip}
              enterDelay={300} key={i}>
              <div>{/* Extra level needed by tooltip when child is disabled */}
                <BusyButton id={b.id} color='secondary' onClick={b.action} disabled={b.disabled}
                  title={b.title} icon={b.icon} busy={b.busy || false}
                  outlined={b.outlined || false} />
              </div>
            </Tooltip>
          ))}
        </div>
      </div>
    );
  }
}

export default Toolbar;
