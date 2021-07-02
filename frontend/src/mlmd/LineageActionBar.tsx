/*
 * Copyright 2021 The Kubeflow Authors
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
import grey from '@material-ui/core/colors/grey';
import Button from '@material-ui/core/Button';
import ArrowRightAltIcon from '@material-ui/icons/ArrowRightAlt';
import ReplayIcon from '@material-ui/icons/Replay';
import { classes, stylesheet } from 'typestyle';
import { color, commonCss, fonts, padding } from './Css';
import { CSSProperties } from 'typestyle/lib/types';
import { getArtifactName, getResourcePropertyViaFallBack } from './Utils';
import { Artifact } from 'src/third_party/mlmd';
import { ArtifactProperties, ArtifactCustomProperties } from './Api';

const baseLinkButton: CSSProperties = {
  backgroundColor: 'transparent',
  border: 'none',
  cursor: 'pointer',
  display: 'inline',
  margin: 0,
  padding: 0,
};

const baseBreadcrumb = {
  ...baseLinkButton,
  fontFamily: fonts.secondary,
  fontWeight: 500,
};

const actionBarCss = stylesheet({
  actionButton: {
    color: color.strong,
  },
  workspace: {
    ...baseBreadcrumb,
    fontStyle: 'italic',
  },
  workspaceSep: {
    display: 'block',
    color: '#3c3c3c',
    $nest: {
      '&::before': {
        content: '""',
        color: '#9f9f9f',
        margin: '0 .75em',
        border: '1px solid',
        background: 'currentColor',
      },
    },
  },
  breadcrumbContainer: {
    alignItems: 'center',
    display: 'flex',
    flexShrink: 1,
    overflow: 'hidden',
  },
  breadcrumbInactive: {
    color: color.grey,
    ...baseBreadcrumb,
    $nest: {
      '&:hover': {
        textDecoration: 'underline',
      },
    },
  },
  breadcrumbActive: {
    color: color.strong,
    ...baseBreadcrumb,
    $nest: {
      '&:hover': {
        cursor: 'default',
      },
    },
  },
  breadcrumbSeparator: {
    color: grey[400],
  },
  container: {
    borderBottom: '1px solid ' + color.separator,
    height: '48px',
    justifyContent: 'space-between',
  },
});

export interface LineageActionBarProps {
  initialTarget?: Artifact;
  setLineageViewTarget(artifact: Artifact): void;
}

export interface LineageActionBarState {
  history: Artifact[];
}

const BreadcrumbSeparator: React.FC = () => (
  <div className={classes(commonCss.flex)}>
    <ArrowRightAltIcon className={classes(actionBarCss.breadcrumbSeparator, padding(10, 'lr'))} />
  </div>
);

/** Shows the current navigation history and actions available to the Lineage Explorer. */
export class LineageActionBar extends React.Component<
  LineageActionBarProps,
  LineageActionBarState
> {
  constructor(props: LineageActionBarProps) {
    super(props);
    this.reset = this.reset.bind(this);
    this.pushHistory = this.pushHistory.bind(this);
    this.state = {
      history: [],
    };
    if (this.props.initialTarget) {
      this.state.history.push(this.props.initialTarget);
    }
  }

  public pushHistory(artifact: Artifact) {
    this.setState({
      history: [...this.state.history, artifact],
    });
  }

  public render() {
    const breadcrumbs: JSX.Element[] = [
      <span className={classes(actionBarCss.workspace)} key='workspace'>
        {getResourcePropertyViaFallBack(
          this.state.history[0],
          [ArtifactProperties, ArtifactCustomProperties],
          ['PIPELINE_NAME', 'WORKSPACE'],
        )}
      </span>,
      <aside key='workspace-sep' className={actionBarCss.workspaceSep} />,
    ];
    this.state.history.forEach((artifact: Artifact, index) => {
      const isActive = index === this.state.history.length - 1;
      const onBreadcrumbClicked = () => {
        this.sliceHistory(index);
      };
      breadcrumbs.push(
        <button
          key={`breadcrumb-${index}`}
          className={classes(
            isActive ? actionBarCss.breadcrumbActive : actionBarCss.breadcrumbInactive,
          )}
          disabled={isActive}
          onClick={onBreadcrumbClicked}
        >
          {getArtifactName(artifact)}
        </button>,
      );
      if (!isActive) {
        breadcrumbs.push(<BreadcrumbSeparator key={`separator-${index}`} />);
      }
    });

    return (
      <div className={classes(actionBarCss.container, padding(25, 'lr'), commonCss.flex)}>
        <div className={classes(actionBarCss.breadcrumbContainer)}>{breadcrumbs}</div>
        <div>
          <Button
            className={classes(actionBarCss.actionButton)}
            disabled={false}
            onClick={this.reset}
          >
            <ReplayIcon /> Reset
          </Button>
        </div>
      </div>
    );
  }

  private sliceHistory(index: number): void {
    const history = this.state.history.slice(0, index + 1);
    const targetArtifact = history[index];
    const onHistoryChanged = this.props.setLineageViewTarget.bind(this, targetArtifact);
    this.setState(
      {
        history,
      },
      onHistoryChanged,
    );
  }

  private reset() {
    this.sliceHistory(0);
  }
}
