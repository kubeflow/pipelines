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
import Button from '@material-ui/core/Button';
import ChevronLeftIcon from '@material-ui/icons/ChevronLeft';
import IconButton from '@material-ui/core/IconButton';
import JobsIcon from '../icons/jobs';
import PipelinesIcon from '../icons/pipelines';
import { Link } from 'react-router-dom';
import { LocalStorage, LocalStorageKey } from '../lib/LocalStorage';
import { RoutePage } from '../components/Router';
import { RouterProps } from 'react-router';
import { classes, stylesheet } from 'typestyle';
import { fontsize, spacing, dimension, commonCss } from '../Css';


export const sideNavColors = {
  bg: '#0f4471',
  fgActive: 'rgb(227, 233, 237, 1)', // #e3e9ed
  fgActiveInvisible: 'rgb(227, 233, 237, 0)',
  fgDefault: '#87a1b8',
  hover: '#3f698d',
  separator: '#41698d',
};

export const css = stylesheet({
  active: {
    backgroundColor: sideNavColors.hover + ' !important',
    color: sideNavColors.fgActive + ' !important',
  },
  button: {
    borderRadius: dimension.base / 2,
    color: sideNavColors.fgDefault,
    display: 'block',
    fontSize: fontsize.medium,
    fontWeight: 'bold',
    height: dimension.base,
    marginBottom: 4,
    marginLeft: 16,
    maxWidth: 186,
    overflow: 'hidden',
    padding: 10,
    textAlign: 'left',
    textTransform: 'none',
    transition: 'max-width 0.3s',
    whiteSpace: 'nowrap',
    width: 186,
  },
  chevron: {
    color: sideNavColors.fgDefault,
    marginLeft: 16,
    padding: 6,
    transition: 'transform 0.3s',
  },
  collapsedButton: {
    marginLeft: spacing.units(-2),
    maxWidth: dimension.base,
    minWidth: dimension.base,
    padding: 10,
  },
  collapsedChevron: {
    transform: 'rotate(180deg)',
  },
  collapsedLabel: {
    // Hide text when collapsing, but do it with a transition
    color: `${sideNavColors.fgActiveInvisible} !important`,
  },
  collapsedRoot: {
    width: '72px !important',
  },
  collapsedSeparator: {
    margin: `12px !important`,
  },
  label: {
    marginLeft: 10,
    transition: 'color 0.3s',
    verticalAlign: 'super',
  },
  root: {
    background: sideNavColors.bg,
    paddingTop: 24,
    transition: 'width 0.3s',
    width: 220,
  },
  separator: {
    border: '0px none transparent',
    borderTop: `1px solid ${sideNavColors.separator}`,
    margin: 12,
  },
});

interface SideNavProps extends RouterProps {
  page: string;
}

interface SideNavState {
  collapsed: boolean;
  manualCollapseState: boolean;
}

class SideNav extends React.Component<SideNavProps, SideNavState> {
  private _AUTO_COLLAPSE_WIDTH = 800;

  constructor(props: any) {
    super(props);

    const collapsed = LocalStorage.isNavbarCollapsed();

    this.state = {
      collapsed,
      manualCollapseState: LocalStorage.hasKey(LocalStorageKey.navbarCollapsed),
    };
  }

  public componentDidMount() {
    window.addEventListener('resize', this._maybeResize.bind(this));
    this._maybeResize();
  }

  public render() {
    const page = this.props.page;
    const { collapsed } = this.state;
    const iconColor = {
      active: sideNavColors.fgActive,
      inactive: sideNavColors.fgDefault,
    };

    return (
      <div id='sideNav' className={classes(css.root, commonCss.noShrink, collapsed && css.collapsedRoot)}>
        <Link id='pipelinesButton' to={RoutePage.PIPELINES} className={commonCss.unstyled}>
          <Button className={classes(css.button,
            page.startsWith(RoutePage.PIPELINES) && css.active,
            collapsed && css.collapsedButton)}>
            <PipelinesIcon color={page.startsWith(RoutePage.PIPELINES) ? iconColor.active : iconColor.inactive} />
            <span className={classes(collapsed && css.collapsedLabel, css.label)}>Pipelines</span>
          </Button>
        </Link>
        <Link id='jobsButton' to={RoutePage.JOBS} className={commonCss.unstyled}>
          <Button className={classes(css.button,
            page.startsWith(RoutePage.JOBS) && css.active,
            collapsed && css.collapsedButton)}>
            <JobsIcon color={page.startsWith(RoutePage.JOBS) ? iconColor.active : iconColor.inactive} />
            <span className={classes(collapsed && css.collapsedLabel, css.label)}>Jobs</span>
          </Button>
        </Link>
        <hr className={classes(css.separator, collapsed && css.collapsedSeparator)} />
        <IconButton className={classes(css.chevron, collapsed && css.collapsedChevron)}
          onClick={this._toggleNavClicked.bind(this)}>
          <ChevronLeftIcon />
        </IconButton>
      </div >
    );
  }

  private _toggleNavClicked() {
    this.setState({
      collapsed: !this.state.collapsed,
      manualCollapseState: true,
    }, () => LocalStorage.saveNavbarCollapsed(this.state.collapsed));
    this._toggleNavCollapsed();
  }

  private _toggleNavCollapsed(shouldCollapse?: boolean): void {
    this.setState({
      collapsed: shouldCollapse !== undefined ? shouldCollapse : !this.state.collapsed,
    });
  }

  private _maybeResize() {
    if (!this.state.manualCollapseState) {
      this._toggleNavCollapsed(window.innerWidth < this._AUTO_COLLAPSE_WIDTH);
    }
  }
}

export default SideNav;
