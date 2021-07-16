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

import { CircularProgress } from '@material-ui/core';
import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import { ExecutionHelpers, getArtifactName, getLinkedArtifactsByEvents } from 'src/mlmd/MlmdUtils';
import { Api, getArtifactTypes } from 'src/mlmd/library';
import {
  ArtifactType,
  Event,
  Execution,
  ExecutionType,
  GetEventsByExecutionIDsRequest,
  GetEventsByExecutionIDsResponse,
  GetExecutionsByIDRequest,
  GetExecutionTypesByIDRequest,
} from 'src/third_party/mlmd';
import { classes, stylesheet } from 'typestyle';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import { RoutePage, RoutePageFactory, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { color, commonCss, padding } from '../Css';
import { logger, serviceErrorToString } from '../lib/Utils';
import { Page, PageErrorHandler } from './Page';

interface ExecutionDetailsState {
  execution?: Execution;
  executionType?: ExecutionType;
  events?: Record<Event.Type, Event[]>;
  artifactTypeMap?: Map<number, ArtifactType>;
}

export default class ExecutionDetails extends Page<{}, ExecutionDetailsState> {
  public state: ExecutionDetailsState = {};

  private get id(): number {
    return parseInt(this.props.match.params[RouteParams.ID], 10);
  }

  public render(): JSX.Element {
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
        />
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Executions', href: RoutePage.EXECUTIONS }],
      pageTitle: `Execution #${this.id}`,
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
          title={'Declared Inputs'}
          events={this.state.events[Event.Type.DECLARED_INPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={'Inputs'}
          events={this.state.events[Event.Type.INPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={'Declared Outputs'}
          events={this.state.events[Event.Type.DECLARED_OUTPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
        <SectionIO
          title={'Outputs'}
          events={this.state.events[Event.Type.OUTPUT]}
          artifactTypeMap={this.state.artifactTypeMap}
        />
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Executions', href: RoutePage.EXECUTIONS }],
      pageTitle: `Execution #${this.props.id} details`,
    };
  }

  private refresh = async (): Promise<void> => {
    await this.load();
  };

  private load = async (): Promise<void> => {
    const metadataStoreServiceClient = Api.getInstance().metadataStoreService;

    // this runs parallelly because it's not a critical resource
    getArtifactTypes(metadataStoreServiceClient)
      .then(artifactTypeMap => {
        this.setState({
          artifactTypeMap,
        });
      })
      .catch(err => {
        this.props.onError('Failed to fetch artifact types', err, 'warning', this.refresh);
      });

    const numberId = this.props.id;
    if (isNaN(numberId) || numberId < 0) {
      const error = new Error(`Invalid execution id: ${this.props.id}`);
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
          `No execution identified by id: ${this.props.id}`,
          undefined,
          'error',
          this.refresh,
        );
        return;
      }

      if (executionResponse.getExecutionsList().length > 1) {
        this.props.onError(
          `Found multiple executions with ID: ${this.props.id}`,
          undefined,
          'error',
          this.refresh,
        );
        return;
      }

      const execution = executionResponse.getExecutionsList()[0];
      this.props.onTitleUpdate(ExecutionHelpers.getName(execution));

      const typeRequest = new GetExecutionTypesByIDRequest();
      typeRequest.setTypeIdsList([execution.getTypeId()]);
      const typeResponse = await metadataStoreServiceClient.getExecutionTypesByID(typeRequest);
      const types = typeResponse.getExecutionTypesList();
      let executionType: ExecutionType | undefined;
      if (!types || types.length === 0) {
        this.props.onError(
          `Cannot find execution type with id: ${execution.getTypeId()}`,
          undefined,
          'error',
          this.refresh,
        );
        return;
      } else if (types.length > 1) {
        this.props.onError(
          `More than one execution type found with id: ${execution.getTypeId()}`,
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
): Record<Event.Type, Event[]> {
  const events: Record<Event.Type, Event[]> = {
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
      events[type].push(event);
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
  events: Event[];
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
    try {
      const linkedArtifacts = await getLinkedArtifactsByEvents(this.props.events);

      const artifactDataMap = {};
      linkedArtifacts.forEach(linkedArtifact => {
        const id = linkedArtifact.event.getArtifactId();
        if (!id) {
          logger.error('Artifact has empty id', linkedArtifact.artifact.toObject());
          return;
        }
        artifactDataMap[id] = {
          id,
          name: getArtifactName(linkedArtifact),
          typeId: linkedArtifact.artifact.getTypeId(),
          uri: linkedArtifact.artifact.getUri() || '',
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
    const { title, events } = this.props;
    if (events.length === 0) {
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
            {events.map(event => {
              const id = event.getArtifactId();
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
  <tr className={css.row}>
    <td className={css.tableCell}>{id}</td>
    <td className={css.tableCell}>
      {id ? (
        <Link className={commonCss.link} to={RoutePageFactory.artifactDetails(id)}>
          {name ? name : '(No name)'}
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
  row: {
    $nest: {
      '&:hover': {
        backgroundColor: color.whiteSmoke,
      },
      '&:hover a': {
        color: color.linkLight,
      },
      a: {
        color: color.link,
      },
    },
  },
});
