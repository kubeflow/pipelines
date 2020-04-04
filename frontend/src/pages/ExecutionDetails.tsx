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
  titleCase,
  ExecutionType,
} from '@kubeflow/frontend';
import { CircularProgress } from '@material-ui/core';
import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import { classes, stylesheet } from 'typestyle';
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { RoutePage, RouteParams, RoutePageFactory } from '../components/Router';
import { commonCss, padding } from '../Css';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import { serviceErrorToString } from '../lib/Utils';
import { GetExecutionTypesByIDRequest } from '@kubeflow/frontend/src/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';

type ArtifactIdList = number[];

interface ExecutionDetailsState {
  execution?: Execution;
  executionType?: ExecutionType;
  events?: Record<Event.Type, ArtifactIdList>;
  artifactTypeMap?: Map<number, ArtifactType>;
}

export default class ExecutionDetails extends Page<{}, ExecutionDetailsState> {
  constructor(props: {}) {
    super(props);
    this.state = {};
    this.load = this.load.bind(this);
  }

  private get fullTypeName(): string {
    // This can be called during constructor, so this.state may not be initialized.
    if (this.state) {
      const { executionType } = this.state;
      const name = executionType?.getName();
      if (name) {
        return name;
      }
    }
    // TODO: remove type from url path
    return this.props.match.params[RouteParams.EXECUTION_TYPE] || '';
  }

  private get id(): string {
    return this.props.match.params[RouteParams.ID];
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public render(): JSX.Element {
    if (!this.state.execution || !this.state.events) {
      return <CircularProgress />;
    }

    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        {
          <ResourceInfo
            resourceType={ResourceType.EXECUTION}
            typeName={this.fullTypeName}
            resource={this.state.execution}
          />
        }
        <SectionIO
          title={'Declared Inputs'}
          artifactIds={this.state.events[Event.Type.DECLARED_INPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={'Inputs'}
          artifactIds={this.state.events[Event.Type.INPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={'Declared Outputs'}
          artifactIds={this.state.events[Event.Type.DECLARED_OUTPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={'Outputs'}
          artifactIds={this.state.events[Event.Type.OUTPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Executions', href: RoutePage.EXECUTIONS }],
      pageTitle: `${this.fullTypeName} ${this.id} details`,
    };
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  private async load(): Promise<void> {
    const metadataStoreServiceClient = Api.getInstance().metadataStoreService;

    // this runs parallelly because it's not a critical resource
    getArtifactTypes(metadataStoreServiceClient)
      .then(artifactTypeMap => {
        this.setState({
          artifactTypeMap,
        });
      })
      .catch(err => {
        this.showPageError('Failed to fetch artifact types', err);
      });

    const numberId = parseInt(this.id, 10);
    if (isNaN(numberId) || numberId < 0) {
      const error = new Error(`Invalid execution id: ${this.id}`);
      this.showPageError(error.message, error);
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
        this.showPageError(`No ${this.fullTypeName} identified by id: ${this.id}`);
      }

      if (executionResponse.getExecutionsList().length > 1) {
        this.showPageError(`Found multiple executions with ID: ${this.id}`);
      }

      const execution = executionResponse.getExecutionsList()[0];
      const executionName =
        getResourceProperty(execution, ExecutionProperties.COMPONENT_ID) ||
        getResourceProperty(execution, ExecutionCustomProperties.TASK_ID, true);
      this.props.updateToolbar({
        pageTitle: executionName ? executionName.toString() : '',
      });

      const typeRequest = new GetExecutionTypesByIDRequest();
      typeRequest.setTypeIdsList([execution.getTypeId()]);
      const typeResponse = await metadataStoreServiceClient.getExecutionTypesByID(typeRequest);
      const types = typeResponse.getExecutionTypesList();
      let executionType: ExecutionType | undefined;
      if (!types || types.length === 0) {
        this.showPageError(`Cannot find execution type with id: ${execution.getTypeId()}`);
      } else if (types.length > 1) {
        this.showPageError(`More than one execution type found with id: ${execution.getTypeId()}`);
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
      this.showPageError(serviceErrorToString(err));
    }
  }
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
      {type && id ? (
        <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(type, id)}>
          {id}
        </Link>
      ) : (
        id
      )}
    </td>
    <td className={css.tableCell}>
      {type && id ? (
        <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(type, id)}>
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
