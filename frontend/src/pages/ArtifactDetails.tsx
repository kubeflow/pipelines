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
  getResourceProperty,
  LineageResource,
  LineageView,
  titleCase,
  ArtifactType,
} from '@kubeflow/frontend';
import { CircularProgress } from '@material-ui/core';
import * as React from 'react';
import { Route, Switch } from 'react-router-dom';
import { classes } from 'typestyle';
import MD2Tabs from '../atoms/MD2Tabs';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import { RoutePage, RoutePageFactory, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { commonCss, padding } from '../Css';
import { logger, serviceErrorToString } from '../lib/Utils';
import { Page, PageProps } from './Page';
import { GetArtifactTypesByIDRequest } from '@kubeflow/frontend/src/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
import { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

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

class ArtifactDetails extends Page<{ t: TFunction }, ArtifactDetailsState> {
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
    return (
      <div className={classes(commonCss.page)}>
        <Switch>
          {/*
           ** This is react-router's nested route feature.
           ** reference: https://reacttraining.com/react-router/web/example/nesting
           */}
          <Route path={this.props.match.path} exact={true}>
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
          <Route path={`${this.props.match.path}/${LINEAGE_PATH}`} exact={true}>
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
    const { t } = this.props;
    return {
      actions: {},
      breadcrumbs: [{ displayName: t('common:artifacts'), href: RoutePage.ARTIFACTS }],
      pageTitle: `${t('artifactNum', { id: this.id })}`,
      t,
    };
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  private load = async (): Promise<void> => {
    const request = new GetArtifactsByIDRequest();
    const { t } = this.props;
    request.setArtifactIdsList([Number(this.id)]);

    try {
      const response = await this.api.metadataStoreService.getArtifactsByID(request);
      if (response.getArtifactsList().length === 0) {
        this.showPageError(`${t('noArtifactsFoundById')}: ${this.id}`);
        return;
      }
      if (response.getArtifactsList().length > 1) {
        this.showPageError(`${t('multiArtifactsFoundById')}: ${this.id}`);
        return;
      }
      const artifact = response.getArtifactsList()[0];
      const typeRequest = new GetArtifactTypesByIDRequest();
      typeRequest.setTypeIdsList([artifact.getTypeId()]);
      const typeResponse = await this.api.metadataStoreService.getArtifactTypesByID(typeRequest);
      const artifactType = typeResponse.getArtifactTypesList()[0] || undefined;

      const artifactName =
        getResourceProperty(artifact, ArtifactProperties.NAME) ||
        getResourceProperty(artifact, ArtifactCustomProperties.NAME, true);
      let title = artifactName ? artifactName.toString() : '';
      const version = getResourceProperty(artifact, ArtifactProperties.VERSION);
      if (version) {
        title += ` (${t('common:version')}: ${version})`;
      }
      this.props.updateToolbar({
        pageTitle: title,
      });
      this.setState({ artifact, artifactType });
    } catch (err) {
      this.showPageError(serviceErrorToString(err));
    }
  };

  private switchTab = (selectedTab: number) => {
    switch (selectedTab) {
      case ArtifactDetailsTab.LINEAGE_EXPLORER:
        return this.props.history.push(`${this.props.match.url}/${LINEAGE_PATH}`);
      case ArtifactDetailsTab.OVERVIEW:
        return this.props.history.push(this.props.match.url);
      default:
        logger.error(`Unknown selected tab ${selectedTab}.`);
    }
  };
}

// This guarantees that each artifact renders a different <ArtifactDetails /> instance.
const EnhancedArtifactDetails = (props: PageProps) => {
  const { t } = useTranslation('common');
  return <ArtifactDetails {...props} key={props.match.params[RouteParams.ID]} t={t} />;
};

export default EnhancedArtifactDetails;
