/*
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an 'AS IS' BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {
  Api,
  Artifact,
  ArtifactCustomProperties,
  ArtifactProperties,
  GetArtifactsByIDRequest,
  LineageView,
  getResourceProperty,
  titleCase,
  LineageResource,
} from '@kubeflow/frontend';
import * as React from 'react';
import { Page, PageProps } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { RoutePage, RoutePageFactory, RouteParams } from '../components/Router';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { CircularProgress } from '@material-ui/core';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import MD2Tabs from '../atoms/MD2Tabs';
import { serviceErrorToString, logger } from '../lib/Utils';
import { Route, Switch } from 'react-router-dom';

export enum ArtifactDetailsTab {
  OVERVIEW = 0,
  LINEAGE_EXPLORER = 1,
}

const TABS = {
  [ArtifactDetailsTab.OVERVIEW]: { name: 'Overview' },
  [ArtifactDetailsTab.LINEAGE_EXPLORER]: { name: 'Lineage Explorer' },
};

const TAB_NAMES = [ArtifactDetailsTab.OVERVIEW, ArtifactDetailsTab.LINEAGE_EXPLORER].map(
  tabConfig => TABS[tabConfig].name,
);

interface ArtifactDetailsState {
  artifact?: Artifact;
}

class ArtifactDetails extends Page<{}, ArtifactDetailsState> {
  private get fullTypeName(): string {
    return this.props.match.params[RouteParams.ARTIFACT_TYPE] || '';
  }

  private get properTypeName(): string {
    const parts = this.fullTypeName.split('/');
    if (!parts.length) {
      return '';
    }
    return titleCase(parts[parts.length - 1]);
  }

  private get id(): number {
    return Number(this.props.match.params[RouteParams.ID]);
  }

  private static buildResourceDetailsPageRoute(
    resource: LineageResource,
    typeName: string,
  ): string {
    let route;
    if (resource instanceof Artifact) {
      route = RoutePageFactory.artifactDetails(typeName, resource.getId());
    } else {
      route = RoutePage.EXECUTION_DETAILS.replace(`:${RouteParams.EXECUTION_TYPE}+`, typeName);
    }
    return route.replace(`:${RouteParams.ID}`, String(resource.getId()));
  }

  public state: ArtifactDetailsState = {};

  private api = Api.getInstance();

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public render(): JSX.Element {
    if (!this.state.artifact) {
      return (
        <div className={commonCss.page}>
          <CircularProgress className={commonCss.absoluteCenter} />
        </div>
      );
    }
    return (
      <div className={classes(commonCss.page)}>
        <Switch>
          <Route path={`${this.props.match.path}/`} exact>
            <>
              <div className={classes(padding(20, 't'))}>
                <MD2Tabs
                  tabs={TAB_NAMES}
                  selectedTab={ArtifactDetailsTab.OVERVIEW}
                  onSwitch={this.switchTab}
                />
              </div>
              <div className={classes(padding(20, 'lr'))}>
                <ResourceInfo
                  resourceType={ResourceType.ARTIFACT}
                  typeName={this.properTypeName}
                  resource={this.state.artifact}
                />
              </div>
            </>
          </Route>
          <Route path={`${this.props.match.path}/lineage`}>
            <>
              <div className={classes(padding(20, 't'))}>
                <MD2Tabs
                  tabs={TAB_NAMES}
                  selectedTab={ArtifactDetailsTab.LINEAGE_EXPLORER}
                  onSwitch={this.switchTab}
                />
              </div>
              <LineageView
                target={this.state.artifact}
                buildResourceDetailsPageRoute={ArtifactDetails.buildResourceDetailsPageRoute}
              />
            </>
          </Route>
        </Switch>
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Artifacts', href: RoutePage.ARTIFACTS }],
      pageTitle: `${this.properTypeName} ${this.id} details`,
    };
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  private load = async (): Promise<void> => {
    const request = new GetArtifactsByIDRequest();
    request.setArtifactIdsList([Number(this.id)]);

    try {
      const response = await this.api.metadataStoreService.getArtifactsByID(request);

      if (response.getArtifactsList().length === 0) {
        this.showPageError(`No ${this.fullTypeName} identified by id: ${this.id}`);
        return;
      }

      if (response.getArtifactsList().length > 1) {
        this.showPageError(`Found multiple artifacts with ID: ${this.id}`);
        return;
      }

      const artifact = response.getArtifactsList()[0];

      const artifactName =
        getResourceProperty(artifact, ArtifactProperties.NAME) ||
        getResourceProperty(artifact, ArtifactCustomProperties.NAME, true);
      let title = artifactName ? artifactName.toString() : '';
      const version = getResourceProperty(artifact, ArtifactProperties.VERSION);
      if (version) {
        title += ` (version: ${version})`;
      }
      this.props.updateToolbar({
        pageTitle: title,
      });
      this.setState({ artifact });
    } catch (err) {
      this.showPageError(serviceErrorToString(err));
    }
  };

  private switchTab = (selectedTab: number) => {
    switch (selectedTab) {
      case ArtifactDetailsTab.LINEAGE_EXPLORER:
        return this.props.history.push(
          `${RoutePageFactory.artifactDetails(this.fullTypeName, this.id)}/lineage`,
        );
      case ArtifactDetailsTab.OVERVIEW:
        return this.props.history.push(
          `${RoutePageFactory.artifactDetails(this.fullTypeName, this.id)}`,
        );
      default:
        logger.error(`Unknown selected tab ${selectedTab}.`);
    }
  };
}

// This guarantees that each artifact renders a different <ArtifactDetails /> instance.
const EnhancedArtifactDetails = (props: PageProps) => {
  return <ArtifactDetails {...props} key={props.match.params[RouteParams.ID]} />;
};

export default EnhancedArtifactDetails;
