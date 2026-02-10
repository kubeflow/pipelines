/*
 * Copyright 2019 The Kubeflow Authors
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
  ArtifactProperties,
  getResourceProperty,
  LineageResource,
  LineageView,
} from 'src/mlmd/library';
import {
  ArtifactType,
  Artifact,
  GetArtifactsByIDRequest,
  GetArtifactTypesByIDRequest,
} from 'src/third_party/mlmd';
import { CircularProgress } from '@mui/material';
import * as React from 'react';
// React Router v6 - Routes not needed here after refactoring
import { classes } from 'typestyle';
import MD2Tabs from '../atoms/MD2Tabs';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import { RoutePage, RoutePageFactory, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { commonCss, padding } from '../Css';
import {
  errorToMessage,
  isServiceError,
  logger,
  serviceErrorToString,
  titleCase,
} from '../lib/Utils';
import { Page, PageProps } from './Page';
import { ArtifactHelpers } from 'src/mlmd/MlmdUtils';

export enum ArtifactDetailsTab {
  OVERVIEW = 0,
  LINEAGE_EXPLORER = 1,
}

const LINEAGE_PATH = 'lineage';

const TABS = {
  [ArtifactDetailsTab.OVERVIEW]: { name: 'Overview' },
  [ArtifactDetailsTab.LINEAGE_EXPLORER]: { name: 'Lineage Explorer' },
};

const TAB_NAMES = [ArtifactDetailsTab.OVERVIEW, ArtifactDetailsTab.LINEAGE_EXPLORER].map(
  tabConfig => TABS[tabConfig].name,
);

interface ArtifactDetailsState {
  artifact?: Artifact;
  artifactType?: ArtifactType;
}

class ArtifactDetails extends Page<{}, ArtifactDetailsState> {
  private get fullTypeName(): string {
    return this.state.artifactType?.getName() || '';
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
    _: string, // typename is no longer used
  ): string {
    // HACK: this distinguishes artifact from execution, only artifacts have
    // the getUri() method.
    // TODO: switch to use typedResource
    if (typeof resource['getUri'] === 'function') {
      return RoutePageFactory.artifactDetails(resource.getId());
    } else {
      return RoutePageFactory.executionDetails(resource.getId());
    }
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
    // Determine which tab to show based on current path
    const isLineageView = this.props.location.pathname.endsWith(`/${LINEAGE_PATH}`);
    const selectedTab = isLineageView ? ArtifactDetailsTab.LINEAGE_EXPLORER : ArtifactDetailsTab.OVERVIEW;

    return (
      <div className={classes(commonCss.page)}>
        <div className={classes(padding(20, 't'))}>
          <MD2Tabs
            tabs={TAB_NAMES}
            selectedTab={selectedTab}
            onSwitch={this.switchTab}
          />
        </div>
        {selectedTab === ArtifactDetailsTab.OVERVIEW ? (
          <div className={classes(padding(20, 'lr'))}>
            <ResourceInfo
              resourceType={ResourceType.ARTIFACT}
              typeName={this.properTypeName}
              resource={this.state.artifact}
            />
          </div>
        ) : (
          <LineageView
            target={this.state.artifact}
            buildResourceDetailsPageRoute={ArtifactDetails.buildResourceDetailsPageRoute}
          />
        )}
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Artifacts', href: RoutePage.ARTIFACTS }],
      pageTitle: `Artifact #${this.id}`,
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
        this.showPageError(`No artifact identified by id: ${this.id}`);
        return;
      }
      if (response.getArtifactsList().length > 1) {
        this.showPageError(`Found multiple artifacts with ID: ${this.id}`);
        return;
      }
      const artifact = response.getArtifactsList()[0];
      const typeRequest = new GetArtifactTypesByIDRequest();
      typeRequest.setTypeIdsList([artifact.getTypeId()]);
      const typeResponse = await this.api.metadataStoreService.getArtifactTypesByID(typeRequest);
      const artifactType = typeResponse.getArtifactTypesList()[0] || undefined;

      let title = ArtifactHelpers.getName(artifact);
      const version = getResourceProperty(artifact, ArtifactProperties.VERSION);
      if (version) {
        title += ` (version: ${version})`;
      }
      this.props.updateToolbar({
        pageTitle: title,
      });
      this.setState({ artifact, artifactType });
    } catch (err) {
      if (isServiceError(err)) {
        this.showPageError(serviceErrorToString(err));
      } else {
        const errorMessage = await errorToMessage(err);
        this.showPageError(
          errorMessage ? `Error: ${errorMessage}` : 'Error: failed to load artifact.',
        );
      }
    }
  };

  private switchTab = (selectedTab: number) => {
    const basePath = this.props.location.pathname.replace(`/${LINEAGE_PATH}`, '');
    switch (selectedTab) {
      case ArtifactDetailsTab.LINEAGE_EXPLORER:
        return this.props.navigate(`${basePath}/${LINEAGE_PATH}`);
      case ArtifactDetailsTab.OVERVIEW:
        return this.props.navigate(basePath);
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
