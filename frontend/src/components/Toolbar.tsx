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
import Tooltip from '@material-ui/core/Tooltip';
import { Link } from 'react-router-dom';
import { classes, stylesheet } from 'typestyle';
import { spacing, fontsize, color, dimension, commonCss } from '../Css';

export interface ToolbarActionConfig {
  action: () => void;
  busy?: boolean;
  disabled: boolean;
  disabledTitle?: string;
  icon: any;
  id: string;
  title: string;
  tooltip: string;
}

const backIconHeight = 28;

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
    paddingRight: 12,
  },
  breadcrumbs: {
    fontSize: fontsize.medium,
    paddingBottom: 6,
    paddingLeft: 40,
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
    marginLeft: 20,
  },
});

export interface ToolbarProps {
  breadcrumbs: Array<{ displayName: string, href: string }>;
  actions: ToolbarActionConfig[];
}

class Toolbar extends React.Component<ToolbarProps> {

  public render() {
    const currentPage = this.props.breadcrumbs.length ?
      this.props.breadcrumbs[this.props.breadcrumbs.length - 1].displayName : '';
    const breadcrumbs = this.props.breadcrumbs.slice(0, this.props.breadcrumbs.length - 1);
    return (
      <div className={css.root}>
        <div>
          {/* Breadcrumb */}
          <div className={classes(css.breadcrumbs, commonCss.flex)}>
            {breadcrumbs.map((crumb, i) => (
              <span className={commonCss.flex} key={i} title={crumb.displayName}>
                {i !==0 && <ChevronRightIcon className={css.chevron} />}
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
              <Tooltip title={breadcrumbs[breadcrumbs.length - 1].displayName} enterDelay={300}>
                <Link className={classes(commonCss.unstyled, css.backLink)}
                    to={breadcrumbs[breadcrumbs.length - 1].href}>
                  <ArrowBackIcon className={css.backIcon}/>
                </Link>
              </Tooltip>}
            {/* Resource Name */}
            <span className={classes(css.pageName, commonCss.ellipsis)} title={currentPage}>
              {currentPage}
            </span>
          </div>
        </div>
        <div className={css.actions}>
          {this.props.actions.map((b, i) => (
            <Tooltip title={(b.disabled && b.disabledTitle) ? b.disabledTitle : b.tooltip}
              enterDelay={300} key={i}>
              <div>{/* Extra level needed by tooltip when child is disabled*/}
                <BusyButton id={b.id} color='secondary' onClick={b.action} disabled={b.disabled}
                  title={b.title} icon={b.icon} busy={b.busy || false} />
              </div>
            </Tooltip>
          ))}
        </div>
      </div>
    );
  }
}

export default Toolbar;
