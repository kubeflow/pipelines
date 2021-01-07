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

import IconButton from '@material-ui/core/IconButton';
import Tooltip from '@material-ui/core/Tooltip';
import ArrowBackIcon from '@material-ui/icons/ArrowBack';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';
import { History } from 'history';
import * as React from 'react';
import { CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import { classes, stylesheet } from 'typestyle';
import BusyButton from '../atoms/BusyButton';
import { color, commonCss, dimension, fonts, fontsize, spacing } from '../Css';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

export interface ToolbarActionMap {
  [key: string]: ToolbarActionConfig;
}

export interface ToolbarActionConfig {
  action: () => void;
  busy?: boolean;
  disabled?: boolean;
  disabledTitle?: string;
  icon?: any;
  id?: string;
  outlined?: boolean;
  primary?: boolean;
  style?: CSSProperties;
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
  disabled: {
    color: '#aaa',
  },
  enabled: {
    color: color.foreground,
  },
  link: {
    $nest: {
      '&:hover': {
        background: color.lightGrey,
      },
    },
    borderRadius: 3,
    padding: 3,
  },
  pageName: {
    color: color.strong,
    fontSize: fontsize.pageTitle,
    lineHeight: '28px',
  },
  root: {
    alignItems: 'center',
    display: 'flex',
    flexShrink: 0,
    height: dimension.jumbo,
    justifyContent: 'space-between',
  },
  topLevelToolbar: {
    borderBottom: `1px solid ${color.lightGrey}`,
    paddingBottom: 15,
    paddingLeft: 20,
  },
});

export interface ToolbarProps {
  actions: ToolbarActionMap;
  breadcrumbs: Breadcrumb[];
  history?: History;
  pageTitle: string | JSX.Element;
  pageTitleTooltip?: string;
  topLevelToolbar?: boolean;
  t: TFunction;
}

class Toolbar extends React.Component<ToolbarProps> {
  public render(): JSX.Element | null {
    const { t } = this.props;

    const { actions, breadcrumbs, pageTitle, pageTitleTooltip } = { ...this.props };

    if (!actions.length && !breadcrumbs.length && !pageTitle) {
      return null;
    }

    return (
      <div
        className={classes(css.root, this.props.topLevelToolbar !== false && css.topLevelToolbar)}
      >
        <div style={{ minWidth: 100 }}>
          {/* Breadcrumb */}
          <div className={classes(css.breadcrumbs, commonCss.flex)}>
            {breadcrumbs.map((crumb, i) => (
              <span className={commonCss.flex} key={i} title={crumb.displayName}>
                {i !== 0 && <ChevronRightIcon className={css.chevron} />}
                <Link
                  className={classes(commonCss.unstyled, commonCss.ellipsis, css.link)}
                  to={crumb.href}
                >
                  {crumb.displayName}
                </Link>
              </span>
            ))}
          </div>
          <div className={commonCss.flex}>
            {/* Back Arrow */}
            {breadcrumbs.length > 0 && (
              <Tooltip title={t('common:back')} enterDelay={300}>
                <div>
                  {' '}
                  {/* Div needed because we sometimes disable a button within a tooltip */}
                  <IconButton
                    className={css.backLink}
                    disabled={this.props.history!.length < 2}
                    onClick={this.props.history!.goBack}
                  >
                    <ArrowBackIcon
                      className={classes(
                        css.backIcon,
                        this.props.history!.length < 2 ? css.disabled : css.enabled,
                      )}
                    />
                  </IconButton>
                </div>
              </Tooltip>
            )}
            {/* Resource Name */}
            <span
              className={classes(css.pageName, commonCss.ellipsis)}
              title={pageTitleTooltip}
              data-testid='page-title' // TODO: use a proper h1 tag for page titles and let tests query this by h1.
            >
              {pageTitle}
            </span>
          </div>
        </div>
        {/* Actions / Buttons */}
        <div className={css.actions}>
          {Object.keys(actions).map((buttonKey, i) => {
            const button = actions[buttonKey];
            return (
              <Tooltip
                title={
                  button.disabled && button.disabledTitle ? button.disabledTitle : button.tooltip
                }
                enterDelay={300}
                key={i}
              >
                <div style={button.style}>
                  {/* Extra level needed by tooltip when child is disabled */}
                  <BusyButton
                    id={button.id}
                    color='secondary'
                    onClick={button.action}
                    disabled={button.disabled}
                    title={button.title}
                    icon={button.icon}
                    busy={button.busy || false}
                    outlined={(button.outlined && !button.primary) || false}
                    className={button.primary ? commonCss.buttonAction : ''}
                  />
                </div>
              </Tooltip>
            );
          })}
        </div>
      </div>
    );
  }
}
export default withTranslation('common')(Toolbar);
