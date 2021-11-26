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

/*
 * Copyright 2019 Google LLC
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
// tslint:disable: object-literal-sort-keys

import groupBy from 'lodash.groupby';
import * as React from 'react';
import { classes, stylesheet } from 'typestyle';
import { commonCss, zIndex } from './Css';
import { LineageCardColumn, CardDetails } from './LineageCardColumn';
import { LineageActionBar } from './LineageActionBar';
import {
  Artifact,
  ArtifactType,
  Event,
  Execution,
  ExecutionType,
  GetArtifactsByIDRequest,
  GetEventsByArtifactIDsRequest,
  GetEventsByExecutionIDsRequest,
  GetExecutionsByIDRequest,
  MetadataStoreServicePromiseClient,
} from 'src/third_party/mlmd';
import { RefObject } from 'react';
import { getArtifactTypes, getExecutionTypes } from './LineageApi';
import { getExecutionTypeName, getTypeName } from './Utils';
import { Api } from './Api';
import { LineageResource } from './LineageTypes';
import CircularProgress from '@material-ui/core/CircularProgress';
import { ArtifactHelpers } from './MlmdUtils';

const isInputEvent = (event: Event) =>
  [Event.Type.INPUT.valueOf(), Event.Type.DECLARED_INPUT.valueOf()].includes(event.getType());
const isOutputEvent = (event: Event) =>
  [Event.Type.OUTPUT.valueOf(), Event.Type.DECLARED_OUTPUT.valueOf()].includes(event.getType());

/** Default size used when columnPadding prop is unset. */
const DEFAULT_COLUMN_PADDING = 40;

export interface LineageViewProps {
  target: Artifact;
  cardWidth?: number;
  columnPadding?: number;
  buildResourceDetailsPageRoute(resource: LineageResource, typeName: string): string;
}

interface LineageViewState {
  loading: boolean;
  // Dynamic value set to 1/5th of the view width.
  columnWidth: number;
  columnNames: string[];
  columnTypes: string[];
  artifactTypes?: Map<number, ArtifactType>;
  executionTypes?: Map<number, ExecutionType>;
  inputArtifacts: Artifact[];
  inputExecutions: Execution[];
  target: Artifact;
  outputExecutions: Execution[];
  outputArtifacts: Artifact[];

  // A Map from output execution id to the list of output artifact ids it should be connected to.
  // Used to construct the reverse bindings from the Output Artifacts column to the Output
  // Executions column. This column is unique because the artifacts point to different
  // executions instead of just one.
  outputExecutionToOutputArtifactMap: Map<number, number[]>;
}

const LINEAGE_VIEW_CSS = stylesheet({
  LineageExplorer: {
    $nest: {
      '&&': { flexFlow: 'row' },
    },
    position: 'relative',
    background: '#F8F8F9',
    zIndex: 0,
  },
});

export class LineageView extends React.Component<LineageViewProps, LineageViewState> {
  private readonly actionBarRef: React.Ref<LineageActionBar>;
  private readonly containerRef: React.RefObject<HTMLDivElement> = React.createRef();
  private readonly metadataStoreService: MetadataStoreServicePromiseClient;
  private artifactTypes: Map<number, ArtifactType>;
  private executionTypes: Map<number, ExecutionType>;

  constructor(props: any) {
    super(props);
    this.metadataStoreService = Api.getInstance().metadataStoreService;
    this.actionBarRef = React.createRef<LineageActionBar>();
    this.state = {
      loading: true,
      columnWidth: 0,
      columnNames: ['Input Artifact', '', 'Target', '', 'Output Artifact'],
      columnTypes: ['ipa', 'ipx', 'target', 'opx', 'opa'],
      inputArtifacts: [],
      inputExecutions: [],
      outputArtifacts: [],
      outputExecutionToOutputArtifactMap: new Map<number, number[]>(),
      outputExecutions: [],
      target: props.target,
    };
    this.loadData = this.loadData.bind(this);
    this.setTargetFromActionBar = this.setTargetFromActionBar.bind(this);
    this.setTargetFromLineageCard = this.setTargetFromLineageCard.bind(this);
    this.setColumnWidth = this.setColumnWidth.bind(this);
    this.loadData(this.props.target.getId());
  }

  public componentDidMount(): void {
    this.setColumnWidth();
    window.addEventListener('resize', this.setColumnWidth);
  }

  public componentWillUnmount(): void {
    window.removeEventListener('resize', this.setColumnWidth);
  }

  public render(): JSX.Element | null {
    if (!this.artifactTypes || !this.state.columnWidth) {
      return (
        // Return an empty page to allow componentDidMount() to measure the flex container.
        <div className={classes(commonCss.page)} ref={this.containerRef} />
      );
    }

    const { columnNames } = this.state;
    const columnPadding = this.props.columnPadding || DEFAULT_COLUMN_PADDING;
    return (
      <div className={classes(commonCss.page)} ref={this.containerRef}>
        <LineageActionBar
          ref={this.actionBarRef}
          initialTarget={this.props.target}
          setLineageViewTarget={this.setTargetFromActionBar}
        />
        <div
          className={classes(commonCss.page, LINEAGE_VIEW_CSS.LineageExplorer, 'LineageExplorer')}
        >
          {this.state.loading && (
            <>
              <div className={commonCss.busyOverlay} />
              <CircularProgress
                size={48}
                className={commonCss.absoluteCenter}
                style={{ zIndex: zIndex.BUSY_OVERLAY }}
              />
            </>
          )}
          <LineageCardColumn
            type='artifact'
            cards={this.buildArtifactCards(this.state.inputArtifacts)}
            title={`${columnNames[0]}`}
            columnWidth={this.state.columnWidth}
            columnPadding={columnPadding}
            setLineageViewTarget={this.setTargetFromLineageCard}
          />
          <LineageCardColumn
            type='execution'
            cards={this.buildExecutionCards(this.state.inputExecutions)}
            columnPadding={columnPadding}
            title={`${columnNames[1]}`}
            columnWidth={this.state.columnWidth}
          />
          <LineageCardColumn
            type='artifact'
            cards={this.buildArtifactCards([this.state.target], /* isTarget= */ true)}
            columnPadding={columnPadding}
            skipEdgeCanvas={true /* Canvas will be drawn by the next canvas's reverse edges. */}
            title={`${columnNames[2]}`}
            columnWidth={this.state.columnWidth}
          />
          <LineageCardColumn
            type='execution'
            cards={this.buildExecutionCards(this.state.outputExecutions)}
            columnPadding={columnPadding}
            reverseBindings={true}
            title={`${columnNames[3]}`}
            columnWidth={this.state.columnWidth}
          />
          <LineageCardColumn
            type='artifact'
            cards={this.buildArtifactCards(this.state.outputArtifacts)}
            connectedCards={this.buildExecutionCards(this.state.outputExecutions)}
            columnPadding={columnPadding}
            columnWidth={this.state.columnWidth}
            outputExecutionToOutputArtifactMap={this.state.outputExecutionToOutputArtifactMap}
            reverseBindings={true}
            setLineageViewTarget={this.setTargetFromLineageCard}
            title={`${columnNames[4]}`}
          />
        </div>
      </div>
    );
  }

  private buildArtifactCards(artifacts: Artifact[], isTarget: boolean = false): CardDetails[] {
    const orderedCardsByType: CardDetails[] = [];

    let currentType: number;
    let currentTypeName: string;
    let currentCard: CardDetails;

    artifacts.forEach(artifact => {
      if (!currentType || artifact.getTypeId() !== currentType) {
        // Create a new card
        currentType = artifact.getTypeId();
        currentTypeName = getTypeName(Number(currentType), this.artifactTypes);
        currentCard = {
          title: currentTypeName,
          elements: [],
        };
        orderedCardsByType.push(currentCard);
      }

      currentCard.elements.push({
        typedResource: { type: 'artifact', resource: artifact },
        resourceDetailsPageRoute: this.props.buildResourceDetailsPageRoute(
          artifact,
          currentTypeName,
        ),
        prev: !isTarget || this.state.inputExecutions.length > 0,
        next: !isTarget || this.state.outputExecutions.length > 0,
      });
    });

    return orderedCardsByType;
  }

  private buildExecutionCards(executions: Execution[]): CardDetails[] {
    const executionsByTypeId = groupBy(executions, e => e.getTypeId());

    return Object.keys(executionsByTypeId).map(typeId => {
      const executionTypeName = getExecutionTypeName(Number(typeId), this.executionTypes);
      const executionsForType = executionsByTypeId[typeId];
      return {
        title: executionTypeName,
        elements: executionsForType.map(execution => ({
          typedResource: {
            type: 'execution',
            resource: execution,
          },
          resourceDetailsPageRoute: this.props.buildResourceDetailsPageRoute(
            execution,
            executionTypeName,
          ),
          prev: true,
          next: true,
        })),
      };
    });
  }

  private async loadData(targetId: number): Promise<string> {
    this.setState({
      loading: true,
    });
    const [targetArtifactEvents, executionTypes, artifactTypes] = await Promise.all([
      this.getArtifactEvents([targetId]),
      getExecutionTypes(this.metadataStoreService),
      getArtifactTypes(this.metadataStoreService),
    ]);

    Object.assign(this, { artifactTypes, executionTypes });

    const outputExecutionIds: number[] = [];
    const inputExecutionIds: number[] = [];

    for (const event of targetArtifactEvents) {
      const executionId = event.getExecutionId();

      if (isOutputEvent(event)) {
        // The input executions column will show executions where the target
        // was an output of the execution.
        inputExecutionIds.push(executionId);
      } else if (isInputEvent(event)) {
        // The output executions column will show executions where the target
        // was an input for the execution.
        outputExecutionIds.push(executionId);
      }
    }

    const [outputExecutions, inputExecutions] = await Promise.all([
      this.getExecutions(outputExecutionIds),
      this.getExecutions(inputExecutionIds),
    ]);

    const [inputExecutionEvents, outputExecutionEvents] = await Promise.all([
      this.getExecutionEvents(inputExecutionIds),
      this.getExecutionEvents(outputExecutionIds),
    ]);

    // Build the list of input artifacts for the input execution
    const inputExecutionInputArtifactIds: number[] = [];
    inputExecutionEvents.forEach(event => {
      if (!isInputEvent(event)) {
        return;
      }

      inputExecutionInputArtifactIds.push(event.getArtifactId());
    });

    // This map ensures that output artifacts are fetched in an order that is sorted by the
    // connected output execution.
    const outputExecutionToOutputArtifactMap: Map<number, number[]> = new Map();

    const outputExecutionOutputArtifactIds: number[] = [];

    outputExecutionEvents.forEach(event => {
      if (!isOutputEvent(event)) {
        return;
      }

      const executionId = event.getExecutionId();
      if (!outputExecutionToOutputArtifactMap.get(executionId)) {
        outputExecutionToOutputArtifactMap.set(executionId, []);
      }

      const artifactId = event.getArtifactId();
      outputExecutionOutputArtifactIds.push(artifactId);

      const artifacts = outputExecutionToOutputArtifactMap.get(executionId);
      if (artifacts) {
        artifacts.push(artifactId);
      }
    });

    const [inputArtifacts, outputArtifacts] = await Promise.all([
      this.getArtifacts(inputExecutionInputArtifactIds),
      this.getArtifacts(outputExecutionOutputArtifactIds),
    ]);

    this.setState({
      loading: false,
      inputArtifacts,
      inputExecutions,
      outputArtifacts,
      outputExecutionToOutputArtifactMap,
      outputExecutions,
    });
    return '';
  }

  // Updates the view and action bar when the target is set from a lineage card.
  private setTargetFromLineageCard(target: Artifact): void {
    const actionBarRefObject = this.actionBarRef as RefObject<LineageActionBar>;
    if (!actionBarRefObject.current) {
      return;
    }

    actionBarRefObject.current.pushHistory(target);
    this.target = target;
  }

  // Updates the view when the target is changed from the action bar.
  private setTargetFromActionBar(target: Artifact): void {
    this.target = target;
  }

  private set target(target: Artifact) {
    this.loadData(target.getId()).then(
      () => {
        // Target column should be updated in the same frame as other loaded data.
        this.setState({
          target,
        });
      },
      error => {
        console.error(
          `Failed to load related data for artifact: ${ArtifactHelpers.getName(target)}. Details:`,
          error,
        );
        this.setState({
          loading: false,
        });
      },
    );
  }

  private async getExecutions(executionIds: number[]): Promise<Execution[]> {
    const request = new GetExecutionsByIDRequest();
    request.setExecutionIdsList(executionIds);

    const response = await this.metadataStoreService.getExecutionsByID(request);
    return response.getExecutionsList();
  }

  private async getExecutionEvents(executionIds: number[]): Promise<Event[]> {
    const request = new GetEventsByExecutionIDsRequest();
    request.setExecutionIdsList(executionIds);

    const response = await this.metadataStoreService.getEventsByExecutionIDs(request);
    return response.getEventsList();
  }

  private async getArtifacts(artifactIds: number[]): Promise<Artifact[]> {
    const request = new GetArtifactsByIDRequest();
    request.setArtifactIdsList(artifactIds);

    const response = await this.metadataStoreService.getArtifactsByID(request);
    return response.getArtifactsList();
  }

  private async getArtifactEvents(artifactIds: number[]): Promise<Event[]> {
    const request = new GetEventsByArtifactIDsRequest();
    request.setArtifactIdsList(artifactIds);

    const response = await this.metadataStoreService.getEventsByArtifactIDs(request);
    return response.getEventsList();
  }

  private setColumnWidth(): void {
    if (!this.containerRef || !this.containerRef.current) {
      return;
    }

    this.setState({
      columnWidth: this.containerRef.current.clientWidth / 5,
    });
  }
}
