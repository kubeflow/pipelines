/*
 * Copyright 2019 The Kubeflow Authors
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

import { CircularProgress } from '@mui/material';
import type * as React from 'react';
import { Link, Route, Switch } from 'react-router-dom';
import {
  ArtifactArtifactType,
  V2beta1Artifact,
  V2beta1ArtifactTask,
} from 'src/apisv2beta1/artifact';
import MD2Tabs from 'src/atoms/MD2Tabs';
import ArtifactPreview from 'src/components/ArtifactPreview';
import DetailsTable, { ValueComponentProps } from 'src/components/DetailsTable';
import { RoutePage, RoutePageFactory, RouteParams } from 'src/components/Router';
import { ToolbarProps } from 'src/components/Toolbar';
import { RuntimeMetricsVisualizations } from 'src/components/viewers/RuntimeMetricsVisualizations';
import { commonCss, padding } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import { KeyValue } from 'src/lib/StaticGraphParser';
import { errorToMessage, formatDateString, logger } from 'src/lib/Utils';
import { getArtifactSessionInfo } from 'src/lib/v2/RuntimeArtifactUtils';
import { Page, PageProps } from 'src/pages/Page';
import { classes } from 'typestyle';

export enum ArtifactDetailsTab {
  OVERVIEW = 0,
  RELATED_TASKS = 1,
}

const RELATED_TASKS_PATH = 'lineage';
const TAB_NAMES = ['Overview', 'Related tasks'];

interface ArtifactDetailsState {
  artifact?: V2beta1Artifact;
  artifactTasks: V2beta1ArtifactTask[];
  hasError?: boolean;
}

class ArtifactDetails extends Page<{}, ArtifactDetailsState> {
  public state: ArtifactDetailsState = { artifactTasks: [] };

  private get id(): string {
    return this.props.match.params[RouteParams.ID];
  }

  public async componentDidMount(): Promise<void> {
    this._isMounted = true;
    await this.load();
  }

  public render(): React.JSX.Element {
    const { artifact, artifactTasks, hasError } = this.state;
    if (!artifact && !hasError) {
      return (
        <div className={commonCss.page}>
          <CircularProgress className={commonCss.absoluteCenter} />
        </div>
      );
    }
    if (!artifact) {
      return <div className={commonCss.page} />;
    }

    return (
      <div className={commonCss.page}>
        <Switch>
          <Route path={this.props.match.path} exact={true}>
            <ArtifactOverview artifact={artifact} onSwitch={this.switchTab} />
          </Route>
          <Route path={`${this.props.match.path}/${RELATED_TASKS_PATH}`} exact={true}>
            <ArtifactRelationships artifactTasks={artifactTasks} onSwitch={this.switchTab} />
          </Route>
        </Switch>
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Artifacts', href: RoutePage.ARTIFACTS }],
      pageTitle: `Artifact ${this.id}`,
    };
  }

  public async refresh(): Promise<void> {
    await this.load();
  }

  private load = async (): Promise<void> => {
    this.setStateSafe({ hasError: false });
    try {
      const [artifact, artifactTasks] = await Promise.all([
        Apis.artifactServiceApiV2.artifact_1(this.id),
        getAllArtifactTasks(this.id),
      ]);
      if (!this._isMounted) {
        return;
      }
      this.props.updateToolbar({ pageTitle: artifact.name || `Artifact ${this.id}` });
      this.setStateSafe({ artifact, artifactTasks, hasError: false });
      this.clearBanner();
    } catch (error) {
      const message = await errorToMessage(error);
      this.setStateSafe({ hasError: true });
      this.showPageError(message || `Error: failed to load artifact ${this.id}.`, error);
    }
  };

  private switchTab = (selectedTab: number) => {
    switch (selectedTab) {
      case ArtifactDetailsTab.RELATED_TASKS:
        this.props.history.push(`${this.props.match.url}/${RELATED_TASKS_PATH}`);
        return;
      case ArtifactDetailsTab.OVERVIEW:
        this.props.history.push(this.props.match.url.replace(`/${RELATED_TASKS_PATH}`, ''));
        return;
      default:
        logger.error(`Unknown selected tab ${selectedTab}.`);
    }
  };
}

function ArtifactOverview({
  artifact,
  onSwitch,
}: {
  artifact: V2beta1Artifact;
  onSwitch: (selectedTab: number) => void;
}) {
  const details: Array<KeyValue<string>> = [
    ['Artifact ID', artifact.artifact_id || '-'],
    ['Name', artifact.name || '-'],
    ['Type', artifact.type || '-'],
    ['Description', artifact.description || '-'],
    ['Namespace', artifact.namespace || '-'],
    ['Created at', formatDateString(artifact.created_at)],
  ];
  if (artifact.number_value !== undefined) {
    details.push(['Value', String(artifact.number_value)]);
  }
  if (artifact.metadata && Object.keys(artifact.metadata).length) {
    details.push(['Metadata', JSON.stringify(artifact.metadata)]);
  }

  const sessionMap = new Map<string, string | undefined>();
  if (artifact.uri) {
    sessionMap.set(artifact.uri, getArtifactSessionInfo(artifact));
  }

  return (
    <>
      <ArtifactTabs selectedTab={ArtifactDetailsTab.OVERVIEW} onSwitch={onSwitch} />
      <div className={classes(padding(20, 'lr'))}>
        <DetailsTable title='Artifact details' fields={details} />
        {artifact.uri && (
          <DetailsTable<string>
            title='Artifact URI'
            fields={[[artifact.name || 'Artifact', artifact.uri]]}
            valueComponent={ArtifactPreview}
            valueComponentProps={{ namespace: artifact.namespace, sessionMap }}
          />
        )}
        {isVisualizableArtifact(artifact) && (
          <RuntimeMetricsVisualizations artifacts={[artifact]} namespace={artifact.namespace} />
        )}
      </div>
    </>
  );
}

function isVisualizableArtifact(artifact: V2beta1Artifact): boolean {
  return (
    artifact.type === ArtifactArtifactType.Metric ||
    artifact.type === ArtifactArtifactType.ClassificationMetric ||
    artifact.type === ArtifactArtifactType.SlicedClassificationMetric ||
    artifact.type === ArtifactArtifactType.HTML ||
    artifact.type === ArtifactArtifactType.Markdown
  );
}

function ArtifactRelationships({
  artifactTasks,
  onSwitch,
}: {
  artifactTasks: V2beta1ArtifactTask[];
  onSwitch: (selectedTab: number) => void;
}) {
  const relationshipMap = new Map<string, V2beta1ArtifactTask>();
  const fields: Array<KeyValue<string>> = artifactTasks.map((artifactTask, index) => {
    const relationshipId = artifactTask.id || `relationship-${index}`;
    relationshipMap.set(relationshipId, artifactTask);
    return [relationshipLabel(artifactTask, index), relationshipId];
  });

  return (
    <>
      <ArtifactTabs selectedTab={ArtifactDetailsTab.RELATED_TASKS} onSwitch={onSwitch} />
      <div className={classes(padding(20, 'lr'))}>
        {fields.length ? (
          <DetailsTable<string>
            title='Producing and consuming tasks'
            fields={fields}
            valueComponent={RelatedTaskLink}
            valueComponentProps={{ relationshipMap }}
          />
        ) : (
          <div className={commonCss.header2}>No related tasks found.</div>
        )}
      </div>
    </>
  );
}

function RelatedTaskLink({
  value,
  relationshipMap,
}: ValueComponentProps<string> & { relationshipMap?: Map<string, V2beta1ArtifactTask> }) {
  const artifactTask = relationshipMap?.get(String(value));
  if (!artifactTask?.run_id) {
    return <>{artifactTask?.task_id || '-'}</>;
  }
  return (
    <Link className={commonCss.link} to={RoutePageFactory.runDetails(artifactTask.run_id)}>
      Run {artifactTask.run_id}
      {artifactTask.task_id ? ` · Task ${artifactTask.task_id}` : ''}
    </Link>
  );
}

function ArtifactTabs({
  selectedTab,
  onSwitch,
}: {
  selectedTab: ArtifactDetailsTab;
  onSwitch: (selectedTab: number) => void;
}) {
  return (
    <div className={classes(padding(20, 't'))}>
      <MD2Tabs tabs={TAB_NAMES} selectedTab={selectedTab} onSwitch={onSwitch} />
    </div>
  );
}

function relationshipLabel(artifactTask: V2beta1ArtifactTask, index: number): string {
  const direction = artifactTask.type === 'OUTPUT' ? 'Produced as' : 'Consumed as';
  return `${direction} ${artifactTask.key || artifactTask.producer?.task_name || index + 1}`;
}

export async function getAllArtifactTasks(artifactId: string): Promise<V2beta1ArtifactTask[]> {
  const artifactTasks: V2beta1ArtifactTask[] = [];
  const seenPageTokens = new Set<string>();
  let pageToken: string | undefined;
  do {
    const response = await Apis.artifactServiceApiV2.artifactTasks(
      undefined,
      undefined,
      [artifactId],
      undefined,
      pageToken,
      100,
      'id asc',
    );
    artifactTasks.push(...(response.artifact_tasks || []));
    pageToken = response.next_page_token || undefined;
    if (pageToken) {
      if (seenPageTokens.has(pageToken)) {
        throw new Error(`Artifact service returned a repeated page token: ${pageToken}`);
      }
      seenPageTokens.add(pageToken);
    }
  } while (pageToken);
  return artifactTasks;
}

const EnhancedArtifactDetails = (props: PageProps) => (
  <ArtifactDetails {...props} key={props.match.params[RouteParams.ID]} />
);

export default EnhancedArtifactDetails;
