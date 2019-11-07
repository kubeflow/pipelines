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

import * as React from 'react';
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { RoutePage, RouteParams } from '../components/Router';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';
import { CircularProgress } from '@material-ui/core';
import { titleCase, getResourceProperty, serviceErrorToString } from '../lib/Utils';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import { Artifact } from '../generated/src/apis/metadata/metadata_store_pb';
import { Apis, ArtifactProperties } from '../lib/Apis';
import { GetArtifactsByIDRequest } from '../generated/src/apis/metadata/metadata_store_service_pb';

interface ArtifactDetailsState {
  artifact?: Artifact;
}

export default class ArtifactDetails extends Page<{}, ArtifactDetailsState> {
  constructor(props: {}) {
    super(props);
    this.state = {};
    this.load = this.load.bind(this);
  }

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

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public render(): JSX.Element {
    if (!this.state.artifact) {
      return <CircularProgress />;
    }
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        {
          <ResourceInfo
            resourceType={ResourceType.ARTIFACT}
            typeName={this.properTypeName}
            resource={this.state.artifact}
          />
        }
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

  private async load(): Promise<void> {
    const getArtifactsRequest = new GetArtifactsByIDRequest();
    getArtifactsRequest.setArtifactIdsList([this.id]);
    Apis.getMetadataServiceClient().getArtifactsByID(getArtifactsRequest, (err, res) => {
      if (err) {
        this.showPageError(serviceErrorToString(err));
        return;
      }

      if (!res || !res.getArtifactsList().length) {
        this.showPageError(`No ${this.fullTypeName} identified by id: ${this.id}`);
        return;
      }

      if (res.getArtifactsList().length > 1) {
        this.showPageError(`Found multiple artifacts with ID: ${this.id}`);
        return;
      }

      const artifact = res.getArtifactsList()[0];

      const artifactName = getResourceProperty(artifact, ArtifactProperties.NAME);
      let title = artifactName ? artifactName.toString() : '';
      const version = getResourceProperty(artifact, ArtifactProperties.VERSION);
      if (version) {
        title += ` (version: ${version})`;
      }
      this.props.updateToolbar({
        pageTitle: title,
      });
      this.setState({ artifact });
    });
  }
}
