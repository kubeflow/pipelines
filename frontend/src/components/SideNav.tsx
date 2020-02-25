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

import Button from '@material-ui/core/Button';
import IconButton from '@material-ui/core/IconButton';
import Tooltip from '@material-ui/core/Tooltip';
import ArchiveIcon from '@material-ui/icons/Archive';
import ArtifactsIcon from '@material-ui/icons/BubbleChart';
import ChevronLeftIcon from '@material-ui/icons/ChevronLeft';
import JupyterhubIcon from '@material-ui/icons/Code';
import DescriptionIcon from '@material-ui/icons/Description';
import OpenInNewIcon from '@material-ui/icons/OpenInNew';
import ExecutionsIcon from '@material-ui/icons/PlayArrow';
import * as React from 'react';
import { RouterProps } from 'react-router';
import { Link } from 'react-router-dom';
import { classes, stylesheet } from 'typestyle';
import { ExternalLinks, RoutePage, RoutePrefix } from '../components/Router';
import { commonCss, fontsize } from '../Css';
import ExperimentsIcon from '../icons/experiments';
import GitHubIcon from '../icons/GitHub-Mark-120px-plus.png';
import PipelinesIcon from '../icons/pipelines';
import { Apis } from '../lib/Apis';
import { Deployments, KFP_FLAGS } from '../lib/Flags';
import { LocalStorage, LocalStorageKey } from '../lib/LocalStorage';
import { logger } from '../lib/Utils';
import { GkeMetadataContext, GkeMetadata } from 'src/lib/GkeMetadata';

export const sideNavColors = {
  bg: '#f8fafb',
  fgActive: '#0d6de7',
  fgActiveInvisible: 'rgb(227, 233, 237, 0)',
  fgDefault: '#9aa0a6',
  hover: '#f1f3f4',
  separator: '#bdc1c6',
  sideNavBorder: '#e8eaed',
};

const COLLAPSED_SIDE_NAV_SIZE = 72;
const EXPANDED_SIDE_NAV_SIZE = 220;

export const css = stylesheet({
  active: {
    color: sideNavColors.fgActive + ' !important',
  },
  button: {
    $nest: {
      '&::hover': {
        backgroundColor: sideNavColors.hover,
      },
    },
    borderRadius: 0,
    color: sideNavColors.fgDefault,
    display: 'block',
    fontSize: fontsize.medium,
    fontWeight: 'bold',
    height: 44,
    marginBottom: 16,
    maxWidth: EXPANDED_SIDE_NAV_SIZE,
    overflow: 'hidden',
    padding: '12px 10px 10px 26px',
    textAlign: 'left',
    textTransform: 'none',
    transition: 'max-width 0.3s',
    whiteSpace: 'nowrap',
    width: EXPANDED_SIDE_NAV_SIZE,
  },
  chevron: {
    color: sideNavColors.fgDefault,
    marginLeft: 16,
    padding: 6,
    transition: 'transform 0.3s',
  },
  collapsedButton: {
    maxWidth: COLLAPSED_SIDE_NAV_SIZE,
    minWidth: COLLAPSED_SIDE_NAV_SIZE,
    padding: '12px 10px 10px 26px',
  },
  collapsedChevron: {
    transform: 'rotate(180deg)',
  },
  collapsedExternalLabel: {
    // Hide text when collapsing, but do it with a transition of both height and
    // opacity
    height: 0,
    opacity: 0,
  },
  collapsedLabel: {
    // Hide text when collapsing, but do it with a transition
    opacity: 0,
  },
  collapsedRoot: {
    width: `${COLLAPSED_SIDE_NAV_SIZE}px !important`,
  },
  collapsedSeparator: {
    margin: '20px !important',
  },
  envMetadata: {
    color: sideNavColors.fgDefault,
    marginBottom: 16,
    marginLeft: 30,
  },
  icon: {
    height: 20,
    width: 20,
  },
  iconImage: {
    opacity: 0.6, // Images are too colorful there by default, reduce their color.
  },
  indicator: {
    borderBottom: '3px solid transparent',
    borderLeft: `3px solid ${sideNavColors.fgActive}`,
    borderTop: '3px solid transparent',
    height: 38,
    left: 0,
    position: 'absolute',
    zIndex: 1,
  },
  indicatorHidden: {
    opacity: 0,
  },
  infoHidden: {
    opacity: 0,
    transition: 'opacity 0s',
    transitionDelay: '0s',
  },
  infoVisible: {
    opacity: 'initial',
    transition: 'opacity 0.2s',
    transitionDelay: '0.3s',
  },
  label: {
    fontSize: fontsize.base,
    letterSpacing: 0.25,
    marginLeft: 20,
    transition: 'opacity 0.3s',
    verticalAlign: 'super',
  },
  link: {
    color: '#77abda',
  },
  openInNewTabIcon: {
    height: 12,
    marginBottom: 8,
    marginLeft: 5,
    width: 12,
  },
  root: {
    background: sideNavColors.bg,
    borderRight: `1px ${sideNavColors.sideNavBorder} solid`,
    paddingTop: 15,
    transition: 'width 0.3s',
    width: EXPANDED_SIDE_NAV_SIZE,
  },
  separator: {
    border: '0px none transparent',
    borderTop: `1px solid ${sideNavColors.separator}`,
    margin: 20,
  },
});

interface DisplayBuildInfo {
  commitHash: string;
  commitUrl: string;
  date: string;
}

interface SideNavProps extends RouterProps {
  page: string;
}

interface SideNavInternalProps extends SideNavProps {
  gkeMetadata: GkeMetadata;
}

interface SideNavState {
  displayBuildInfo?: DisplayBuildInfo;
  collapsed: boolean;
  jupyterHubAvailable: boolean;
  manualCollapseState: boolean;
}

export class SideNav extends React.Component<SideNavInternalProps, SideNavState> {
  private _isMounted = true;
  private readonly _AUTO_COLLAPSE_WIDTH = 800;
  private readonly _HUB_ADDRESS = '/hub/';

  constructor(props: any) {
    super(props);

    const collapsed = LocalStorage.isNavbarCollapsed();

    this.state = {
      collapsed,
      // Set jupyterHubAvailable to false so UI don't show Jupyter Hub link
      jupyterHubAvailable: false,
      manualCollapseState: LocalStorage.hasKey(LocalStorageKey.navbarCollapsed),
    };
  }

  public async componentDidMount(): Promise<void> {
    window.addEventListener('resize', this._maybeResize.bind(this));
    this._maybeResize();

    async function fetchBuildInfo() {
      const buildInfo = await Apis.getBuildInfo();
      const commitHash = buildInfo.apiServerCommitHash || buildInfo.frontendCommitHash || '';
      return {
        commitHash: commitHash ? commitHash.substring(0, 7) : 'unknown',
        commitUrl:
          'https://www.github.com/kubeflow/pipelines' + (commitHash ? `/commit/${commitHash}` : ''),
        date: buildInfo.buildDate
          ? new Date(buildInfo.buildDate).toLocaleDateString('en-US')
          : 'unknown',
      };
    }
    const [displayBuildInfo, gkeMetadata] = await Promise.all([
      fetchBuildInfo().catch(err => {
        logger.error('Failed to retrieve build info', err);
        return undefined;
      }),
    ]);

    this.setStateSafe({ displayBuildInfo });
  }

  public componentWillUnmount(): void {
    this._isMounted = false;
  }

  public render(): JSX.Element {
    const page = this.props.page;
    const { collapsed, displayBuildInfo } = this.state;
    const { gkeMetadata } = this.props;
    const iconColor = {
      active: sideNavColors.fgActive,
      inactive: sideNavColors.fgDefault,
    };

    return (
      <div
        id='sideNav'
        className={classes(
          css.root,
          commonCss.flexColumn,
          commonCss.noShrink,
          collapsed && css.collapsedRoot,
        )}
      >
        <div style={{ flexGrow: 1 }}>
          {KFP_FLAGS.DEPLOYMENT === Deployments.MARKETPLACE && (
            <>
              <div
                className={classes(
                  css.indicator,
                  !page.startsWith(RoutePage.START) && css.indicatorHidden,
                )}
              />
              <Tooltip
                title={'Getting Started'}
                enterDelay={300}
                placement={'right-start'}
                disableFocusListener={!collapsed}
                disableHoverListener={!collapsed}
                disableTouchListener={!collapsed}
              >
                <Link id='gettingStartedBtn' to={RoutePage.START} className={commonCss.unstyled}>
                  <Button
                    className={classes(
                      css.button,
                      page.startsWith(RoutePage.START) && css.active,
                      collapsed && css.collapsedButton,
                    )}
                  >
                    <DescriptionIcon style={{ width: 20, height: 20 }} />
                    <span className={classes(collapsed && css.collapsedLabel, css.label)}>
                      Getting Started
                    </span>
                  </Button>
                </Link>
              </Tooltip>
            </>
          )}
          <div
            className={classes(
              css.indicator,
              !page.startsWith(RoutePage.PIPELINES) && css.indicatorHidden,
            )}
          />
          <Tooltip
            title={'Pipeline List'}
            enterDelay={300}
            placement={'right-start'}
            disableFocusListener={!collapsed}
            disableHoverListener={!collapsed}
            disableTouchListener={!collapsed}
          >
            <Link id='pipelinesBtn' to={RoutePage.PIPELINES} className={commonCss.unstyled}>
              <Button
                className={classes(
                  css.button,
                  page.startsWith(RoutePage.PIPELINES) && css.active,
                  collapsed && css.collapsedButton,
                )}
              >
                <PipelinesIcon
                  color={
                    page.startsWith(RoutePage.PIPELINES) ? iconColor.active : iconColor.inactive
                  }
                />
                <span className={classes(collapsed && css.collapsedLabel, css.label)}>
                  Pipelines
                </span>
              </Button>
            </Link>
          </Tooltip>
          <div
            className={classes(
              css.indicator,
              !this._highlightExperimentsButton(page) && css.indicatorHidden,
            )}
          />
          <Tooltip
            title={'Experiment List'}
            enterDelay={300}
            placement={'right-start'}
            disableFocusListener={!collapsed}
            disableHoverListener={!collapsed}
            disableTouchListener={!collapsed}
          >
            <Link id='experimentsBtn' to={RoutePage.EXPERIMENTS} className={commonCss.unstyled}>
              <Button
                className={classes(
                  css.button,
                  this._highlightExperimentsButton(page) && css.active,
                  collapsed && css.collapsedButton,
                )}
              >
                <ExperimentsIcon
                  color={
                    this._highlightExperimentsButton(page) ? iconColor.active : iconColor.inactive
                  }
                />
                <span className={classes(collapsed && css.collapsedLabel, css.label)}>
                  Experiments
                </span>
              </Button>
            </Link>
          </Tooltip>
          <div
            className={classes(
              css.indicator,
              !this._highlightArtifactsButton(page) && css.indicatorHidden,
            )}
          />
          <Tooltip
            title={'Artifacts List'}
            enterDelay={300}
            placement={'right-start'}
            disableFocusListener={!collapsed}
            disableHoverListener={!collapsed}
            disableTouchListener={!collapsed}
          >
            <Link id='artifactsBtn' to={RoutePage.ARTIFACTS} className={commonCss.unstyled}>
              <Button
                className={classes(
                  css.button,
                  this._highlightArtifactsButton(page) && css.active,
                  collapsed && css.collapsedButton,
                )}
              >
                <ArtifactsIcon />
                <span className={classes(collapsed && css.collapsedLabel, css.label)}>
                  Artifacts
                </span>
              </Button>
            </Link>
          </Tooltip>
          <div
            className={classes(
              css.indicator,
              !this._highlightExecutionsButton(page) && css.indicatorHidden,
            )}
          />
          <Tooltip
            title={'Executions List'}
            enterDelay={300}
            placement={'right-start'}
            disableFocusListener={!collapsed}
            disableHoverListener={!collapsed}
            disableTouchListener={!collapsed}
          >
            <Link id='executionsBtn' to={RoutePage.EXECUTIONS} className={commonCss.unstyled}>
              <Button
                className={classes(
                  css.button,
                  this._highlightExecutionsButton(page) && css.active,
                  collapsed && css.collapsedButton,
                )}
              >
                <ExecutionsIcon />
                <span className={classes(collapsed && css.collapsedLabel, css.label)}>
                  Executions
                </span>
              </Button>
            </Link>
          </Tooltip>
          {this.state.jupyterHubAvailable && (
            <Tooltip
              title={'Open Jupyter Notebook'}
              enterDelay={300}
              placement={'right-start'}
              disableFocusListener={!collapsed}
              disableHoverListener={!collapsed}
              disableTouchListener={!collapsed}
            >
              <a
                id='jupyterhubBtn'
                href={this._HUB_ADDRESS}
                className={commonCss.unstyled}
                target='_blank'
                rel='noopener'
              >
                <Button className={classes(css.button, collapsed && css.collapsedButton)}>
                  <JupyterhubIcon style={{ height: 20, width: 20 }} />
                  <span className={classes(collapsed && css.collapsedLabel, css.label)}>
                    Notebooks
                  </span>
                  <OpenInNewIcon className={css.openInNewTabIcon} />
                </Button>
              </a>
            </Tooltip>
          )}
          <hr className={classes(css.separator, collapsed && css.collapsedSeparator)} />
          <div
            className={classes(css.indicator, page !== RoutePage.ARCHIVE && css.indicatorHidden)}
          />
          <Tooltip
            title={'Archive'}
            enterDelay={300}
            placement={'right-start'}
            disableFocusListener={!collapsed}
            disableHoverListener={!collapsed}
            disableTouchListener={!collapsed}
          >
            <Link id='archiveBtn' to={RoutePage.ARCHIVE} className={commonCss.unstyled}>
              <Button
                className={classes(
                  css.button,
                  page === RoutePage.ARCHIVE && css.active,
                  collapsed && css.collapsedButton,
                )}
              >
                <ArchiveIcon style={{ height: 20, width: 20 }} />
                <span className={classes(collapsed && css.collapsedLabel, css.label)}>Archive</span>
              </Button>
            </Link>
          </Tooltip>
          <hr className={classes(css.separator, collapsed && css.collapsedSeparator)} />
          <ExternalUri
            title={'Documentation'}
            to={ExternalLinks.DOCUMENTATION}
            collapsed={collapsed}
            icon={className => <DescriptionIcon className={className} />}
          />
          <ExternalUri
            title={'Github Repo'}
            to={ExternalLinks.GITHUB}
            collapsed={collapsed}
            icon={className => (
              <img src={GitHubIcon} className={classes(className, css.iconImage)} alt='Github' />
            )}
          />
          <ExternalUri
            title={'AI Hub Samples'}
            to={ExternalLinks.AI_HUB}
            collapsed={collapsed}
            icon={className => (
              <img
                src='https://www.gstatic.com/aihub/aihub_favicon.png'
                className={classes(className, css.iconImage)}
                alt='AI Hub'
              />
            )}
          />
          <hr className={classes(css.separator, collapsed && css.collapsedSeparator)} />
          <IconButton
            className={classes(css.chevron, collapsed && css.collapsedChevron)}
            onClick={this._toggleNavClicked.bind(this)}
          >
            <ChevronLeftIcon />
          </IconButton>
        </div>
        <div className={collapsed ? css.infoHidden : css.infoVisible}>
          {gkeMetadata.clusterName && gkeMetadata.projectId && (
            <Tooltip
              title={`Cluster name: ${gkeMetadata.clusterName}, Project ID: ${gkeMetadata.projectId}`}
              enterDelay={300}
              placement='top-start'
            >
              <div className={css.envMetadata}>
                <span>Cluster name: </span>
                <a
                  href={`https://console.cloud.google.com/kubernetes/list?project=${gkeMetadata.projectId}&filter=name:${gkeMetadata.clusterName}`}
                  className={classes(css.link, commonCss.unstyled)}
                  rel='noopener'
                  target='_blank'
                >
                  {gkeMetadata.clusterName}
                </a>
              </div>
            </Tooltip>
          )}
          {displayBuildInfo && (
            <Tooltip
              title={'Build date: ' + displayBuildInfo.date}
              enterDelay={300}
              placement={'top-start'}
            >
              <div className={css.envMetadata}>
                <span>Build commit: </span>
                <a
                  href={displayBuildInfo.commitUrl}
                  className={classes(css.link, commonCss.unstyled)}
                  rel='noopener'
                  target='_blank'
                >
                  {displayBuildInfo.commitHash}
                </a>
              </div>
            </Tooltip>
          )}
          <Tooltip title='Report an Issue' enterDelay={300} placement={'top-start'}>
            <div className={css.envMetadata}>
              <a
                href='https://github.com/kubeflow/pipelines/issues/new?template=BUG_REPORT.md'
                className={classes(css.link, commonCss.unstyled)}
                rel='noopener'
                target='_blank'
              >
                Report an Issue
              </a>
            </div>
          </Tooltip>
        </div>
      </div>
    );
  }

  private _highlightExperimentsButton(page: string): boolean {
    return (
      page.startsWith(RoutePage.EXPERIMENTS) ||
      page.startsWith(RoutePage.RUNS) ||
      page.startsWith(RoutePrefix.RECURRING_RUN) ||
      page.startsWith(RoutePage.COMPARE)
    );
  }

  private _highlightArtifactsButton(page: string): boolean {
    return page.startsWith(RoutePrefix.ARTIFACT);
  }

  private _highlightExecutionsButton(page: string): boolean {
    return page.startsWith(RoutePrefix.EXECUTION);
  }

  private _toggleNavClicked(): void {
    this.setStateSafe(
      {
        collapsed: !this.state.collapsed,
        manualCollapseState: true,
      },
      () => LocalStorage.saveNavbarCollapsed(this.state.collapsed),
    );
    this._toggleNavCollapsed();
  }

  private _toggleNavCollapsed(shouldCollapse?: boolean): void {
    this.setStateSafe({
      collapsed: shouldCollapse !== undefined ? shouldCollapse : !this.state.collapsed,
    });
  }

  private _maybeResize(): void {
    if (!this.state.manualCollapseState) {
      this._toggleNavCollapsed(window.innerWidth < this._AUTO_COLLAPSE_WIDTH);
    }
  }

  private setStateSafe(newState: Partial<SideNavState>, cb?: () => void): void {
    if (this._isMounted) {
      this.setState(newState as any, cb);
    }
  }
}

interface ExternalUriProps {
  title: string;
  to: string;
  collapsed: boolean;
  icon: (className: string) => React.ReactNode;
}

// tslint:disable-next-line:variable-name
const ExternalUri: React.FC<ExternalUriProps> = ({ title, to, collapsed, icon }) => (
  <Tooltip
    title={title}
    enterDelay={300}
    placement={'right-start'}
    disableFocusListener={!collapsed}
    disableHoverListener={!collapsed}
    disableTouchListener={!collapsed}
  >
    <a href={to} className={commonCss.unstyled} target='_blank' rel='noopener noreferrer'>
      <Button className={classes(css.button, collapsed && css.collapsedButton)}>
        {icon(css.icon)}
        <span className={classes(collapsed && css.collapsedLabel, css.label)}>{title}</span>
        <OpenInNewIcon className={css.openInNewTabIcon} />
      </Button>
    </a>
  </Tooltip>
);

const EnhancedSideNav: React.FC<SideNavProps> = props => {
  return (
    <GkeMetadataContext.Consumer>
      {value => <SideNav {...props} gkeMetadata={value} />}
    </GkeMetadataContext.Consumer>
  );
};
export default EnhancedSideNav;
