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
  ArtifactCustomProperties,
  ArtifactProperties,
  ArtifactType,
  Event,
  Execution,
  ExecutionCustomProperties,
  ExecutionProperties,
  GetArtifactsByIDRequest,
  GetExecutionsByIDRequest,
  GetEventsByExecutionIDsRequest,
  GetEventsByExecutionIDsResponse,
  getArtifactTypes,
  getResourceProperty,
  logger,
  ExecutionType,
} from '@kubeflow/frontend';
import { CircularProgress } from '@material-ui/core';
import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import { classes, stylesheet } from 'typestyle';
import { Page, PageErrorHandler } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { RoutePage, RouteParams, RoutePageFactory } from '../components/Router';
import { commonCss, padding } from '../Css';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import { serviceErrorToString } from '../lib/Utils';
import { GetExecutionTypesByIDRequest } from '@kubeflow/frontend/src/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

type ArtifactIdList = number[];

interface ExecutionDetailsState {
  execution?: Execution;
  executionType?: ExecutionType;
  events?: Record<Event.Type, ArtifactIdList>;
  artifactTypeMap?: Map<number, ArtifactType>;
}

class ExecutionDetails extends Page<{ t: TFunction }, ExecutionDetailsState> {
  public state: ExecutionDetailsState = {};

  private get id(): number {
    return parseInt(this.props.match.params[RouteParams.ID], 10);
  }

  public render(): JSX.Element {
    const { t } = this.props;
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <ExecutionDetailsContent
          key={this.id}
          id={this.id}
          onError={this.showPageError.bind(this)}
          onTitleUpdate={title =>
            this.props.updateToolbar({
              pageTitle: title,
            })
          }
          t={t}
        />
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    const { t } = this.props;
    return {
      actions: {},
      breadcrumbs: [{ displayName: t('common:executions'), href: RoutePage.EXECUTIONS }],
      pageTitle: `${this.id} ${t('common:details')}`,
      t,
    };
  }

  public async refresh(): Promise<void> {
    // do nothing
  }
}

interface ExecutionDetailsContentProps {
  id: number;
  onError: PageErrorHandler;
  onTitleUpdate: (title: string) => void;
  t: TFunction;
}
export class ExecutionDetailsContent extends Component<
  ExecutionDetailsContentProps,
  ExecutionDetailsState
> {
  public state: ExecutionDetailsState = {};

  private get fullTypeName(): string {
    return this.state.executionType?.getName() || '';
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public render(): JSX.Element {
    const { t } = this.props;
    if (!this.state.execution || !this.state.events) {
      return <CircularProgress />;
    }

    return (
      <div>
        {
          <ResourceInfo
            resourceType={ResourceType.EXECUTION}
            typeName={this.fullTypeName}
            resource={this.state.execution}
          />
        }
        <SectionIO
          title={t('declaredInputs')}
          artifactIds={this.state.events[Event.Type.DECLARED_INPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={t('inputs')}
          artifactIds={this.state.events[Event.Type.INPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={t('declaredOutputs')}
          artifactIds={this.state.events[Event.Type.DECLARED_OUTPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={t('outputs')}
          artifactIds={this.state.events[Event.Type.OUTPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    const { t } = this.props;
    return {
      actions: {},
      breadcrumbs: [{ displayName: t('common:executions'), href: RoutePage.EXECUTIONS }],
      pageTitle: `${t('executionNum')}${this.props.id} ${t('common:details')}`,
      t,
    };
  }

  private refresh = async (): Promise<void> => {
    await this.load();
  };

  private load = async (): Promise<void> => {
    const metadataStoreServiceClient = Api.getInstance().metadataStoreService;
    const { t } = this.props;

    // this runs parallelly because it's not a critical resource
    getArtifactTypes(metadataStoreServiceClient)
      .then(artifactTypeMap => {
        this.setState({
          artifactTypeMap,
        });
      })
      .catch(err => {
        this.props.onError(t('fetchArtifactTypesFailed'), err, 'warning', this.refresh);
      });

    const numberId = this.props.id;
    if (isNaN(numberId) || numberId < 0) {
      const error = new Error(`${t('invalidExecutionId')}: ${this.props.id}`);
      this.props.onError(error.message, error, 'error', this.refresh);
      return;
    }

    const getExecutionsRequest = new GetExecutionsByIDRequest();
    getExecutionsRequest.setExecutionIdsList([numberId]);
    const getEventsRequest = new GetEventsByExecutionIDsRequest();
    getEventsRequest.setExecutionIdsList([numberId]);

    try {
      const [executionResponse, eventResponse] = await Promise.all([
        metadataStoreServiceClient.getExecutionsByID(getExecutionsRequest),
        metadataStoreServiceClient.getEventsByExecutionIDs(getEventsRequest),
      ]);

      if (!executionResponse.getExecutionsList().length) {
        this.props.onError(
          `${t('noExecutionsFoundById')}: ${this.props.id}`,
          undefined,
          'error',
          this.refresh,
        );
        return;
      }

      if (executionResponse.getExecutionsList().length > 1) {
        this.props.onError(
          `${t('multiExecutionsFoundById')}: ${this.props.id}`,
          undefined,
          'error',
          this.refresh,
        );
        return;
      }

      const execution = executionResponse.getExecutionsList()[0];
      const executionName =
        getResourceProperty(execution, ExecutionProperties.COMPONENT_ID) ||
        getResourceProperty(execution, ExecutionCustomProperties.TASK_ID, true);
      this.props.onTitleUpdate(executionName ? executionName.toString() : '');

      const typeRequest = new GetExecutionTypesByIDRequest();
      typeRequest.setTypeIdsList([execution.getTypeId()]);
      const typeResponse = await metadataStoreServiceClient.getExecutionTypesByID(typeRequest);
      const types = typeResponse.getExecutionTypesList();
      let executionType: ExecutionType | undefined;
      if (!types || types.length === 0) {
        this.props.onError(
          `${t('noExecutionTypeFoundById')}: ${execution.getTypeId()}`,
          undefined,
          'error',
          this.refresh,
        );
        return;
      } else if (types.length > 1) {
        this.props.onError(
          `${t('multiExecutionTypeFoundById')}: ${execution.getTypeId()}`,
          undefined,
          'error',
          this.refresh,
        );
        return;
      } else {
        executionType = types[0];
      }

      const events = parseEventsByType(eventResponse);

      this.setState({
        events,
        execution,
        executionType,
      });
    } catch (err) {
      this.props.onError(serviceErrorToString(err), err, 'error', this.refresh);
    }
  };
}

function parseEventsByType(
  response: GetEventsByExecutionIDsResponse | null,
): Record<Event.Type, ArtifactIdList> {
  const events: Record<Event.Type, ArtifactIdList> = {
    [Event.Type.UNKNOWN]: [],
    [Event.Type.DECLARED_INPUT]: [],
    [Event.Type.INPUT]: [],
    [Event.Type.DECLARED_OUTPUT]: [],
    [Event.Type.OUTPUT]: [],
    [Event.Type.INTERNAL_INPUT]: [],
    [Event.Type.INTERNAL_OUTPUT]: [],
  };

  if (!response) {
    return events;
  }

  response.getEventsList().forEach(event => {
    const type = event.getType();
    const id = event.getArtifactId();
    if (type != null && id != null) {
      events[type].push(id);
    }
  });

  return events;
}

interface ArtifactInfo {
  id: number;
  name: string;
  typeId?: number;
  uri: string;
}

interface SectionIOProps {
  title: string;
  artifactIds: number[];
  artifactTypeMap?: Map<number, ArtifactType>;
}
class SectionIO extends Component<
  SectionIOProps,
  { artifactDataMap: { [id: number]: ArtifactInfo } }
> {
  constructor(props: any) {
    super(props);

    this.state = {
      artifactDataMap: {},
    };
  }

  public async componentDidMount(): Promise<void> {
    // loads extra metadata about artifacts
    const request = new GetArtifactsByIDRequest();
    request.setArtifactIdsList(this.props.artifactIds);

    try {
      const response = await Api.getInstance().metadataStoreService.getArtifactsByID(request);

      const artifactDataMap = {};
      response.getArtifactsList().forEach(artifact => {
        const id = artifact.getId();
        if (!id) {
          logger.error('Artifact has empty id', artifact.toObject());
          return;
        }
        artifactDataMap[id] = {
          id,
          name: (getResourceProperty(artifact, ArtifactProperties.NAME) ||
            getResourceProperty(artifact, ArtifactCustomProperties.NAME, true) ||
            '') as string, // TODO: assert name is string
          typeId: artifact.getTypeId(),
          uri: artifact.getUri() || '',
        };
      });
      this.setState({
        artifactDataMap,
      });
    } catch (err) {
      return;
    }
  }

  public render(): JSX.Element | null {
    const { title, artifactIds } = this.props;
    if (artifactIds.length === 0) {
      return null;
    }

    return (
      <section>
        <h2 className={commonCss.header2}>{title}</h2>
        <table>
          <thead>
            <tr>
              <th className={css.tableCell}>Artifact ID</th>
              <th className={css.tableCell}>Name</th>
              <th className={css.tableCell}>Type</th>
              <th className={css.tableCell}>URI</th>
            </tr>
          </thead>
          <tbody>
            {artifactIds.map(id => {
              const data = this.state.artifactDataMap[id] || {};
              const type =
                this.props.artifactTypeMap && data.typeId
                  ? this.props.artifactTypeMap.get(data.typeId)
                  : null;
              return (
                <ArtifactRow
                  key={id}
                  id={id}
                  name={data.name || ''}
                  type={type ? type.getName() : undefined}
                  uri={data.uri}
                />
              );
            })}
          </tbody>
        </table>
      </section>
    );
  }
}

// tslint:disable-next-line:variable-name
const ArtifactRow: React.FC<{ id: number; name: string; type?: string; uri: string }> = ({
  id,
  name,
  type,
  uri,
}) => (
  <tr>
    <td className={css.tableCell}>
      {id ? (
        <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(id)}>
          {id}
        </Link>
      ) : (
        id
      )}
    </td>
    <td className={css.tableCell}>
      {id ? (
        <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(id)}>
          {name}
        </Link>
      ) : (
        name
      )}
    </td>
    <td className={css.tableCell}>{type}</td>
    <td className={css.tableCell}>{uri}</td>
  </tr>
);

const css = stylesheet({
  tableCell: {
    padding: 6,
    textAlign: 'left',
  },
});

export default withTranslation(['executions', 'common'])(ExecutionDetails);
