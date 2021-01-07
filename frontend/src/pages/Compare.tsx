/*
 * Copyright 2018 Google LLC
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
import { Redirect } from 'react-router-dom';
import { useNamespaceChangeEvent } from 'src/lib/KubeflowClient';
import { classes, stylesheet } from 'typestyle';
import { Workflow } from '../../third_party/argo-ui/argo_template';
import { ApiRunDetail } from '../apis/run';
import Hr from '../atoms/Hr';
import Separator from '../atoms/Separator';
import CollapseButton from '../components/CollapseButton';
import CompareTable, { CompareTableProps } from '../components/CompareTable';
import PlotCard, { PlotCardProps } from '../components/PlotCard';
import { QUERY_PARAMS, RoutePage } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { PlotType, ViewerConfig } from '../components/viewers/Viewer';
import { componentMap } from '../components/viewers/ViewerContainer';
import { commonCss, padding } from '../Css';
import { Apis } from '../lib/Apis';
import Buttons from '../lib/Buttons';
import CompareUtils from '../lib/CompareUtils';
import { OutputArtifactLoader } from '../lib/OutputArtifactLoader';
import { URLParser } from '../lib/URLParser';
import { logger } from '../lib/Utils';
import WorkflowParser from '../lib/WorkflowParser';
import { Page, PageProps } from './Page';
import RunList from './RunList';
import { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

const css = stylesheet({
  outputsRow: {
    marginLeft: 15,
    overflowX: 'auto',
  },
});

export interface TaggedViewerConfig {
  config: ViewerConfig;
  runId: string;
  runName: string;
}

export interface CompareState {
  collapseSections: { [key: string]: boolean };
  fullscreenViewerConfig: PlotCardProps | null;
  paramsCompareProps: CompareTableProps;
  metricsCompareProps: CompareTableProps;
  runs: ApiRunDetail[];
  selectedIds: string[];
  viewersMap: Map<PlotType, TaggedViewerConfig[]>;
  workflowObjects: Workflow[];
}

const overviewSectionName = 'experiments:runOverview';
const paramsSectionName = 'experiments:parameters';
const metricsSectionName = 'experiments:metrics';

class Compare extends Page<{ t: TFunction }, CompareState> {
  constructor(props: any) {
    super(props);

    this.state = {
      collapseSections: {},
      fullscreenViewerConfig: null,
      metricsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      paramsCompareProps: { rows: [], xLabels: [], yLabels: [] },
      runs: [],
      selectedIds: [],
      viewersMap: new Map(),
      workflowObjects: [],
    };
  }

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    const { t } = this.props;
    return {
      actions: buttons
        .expandSections(() => this.setState({ collapseSections: {} }))
        .collapseSections(this._collapseAllSections.bind(this))
        .refresh(this.refresh.bind(this))
        .getToolbarActionMap(),
      breadcrumbs: [{ displayName: t('common:experiments'), href: RoutePage.EXPERIMENTS }],
      pageTitle: t('compareRuns'),
      t,
    };
  }

  public render(): JSX.Element {
    const { t } = this.props;
    const { collapseSections, selectedIds, viewersMap } = this.state;
    const queryParamRunIds = new URLParser(this.props).get(QUERY_PARAMS.runlist);
    const runIds = queryParamRunIds ? queryParamRunIds.split(',') : [];

    const runsPerViewerType = (viewerType: PlotType) => {
      return viewersMap.get(viewerType)
        ? viewersMap.get(viewerType)!.filter(el => selectedIds.indexOf(el.runId) > -1)
        : [];
    };

    const viewerDisplayNameEntries = Object.entries(componentMap).map(([key, props]) => [
      key,
      t(props.displayNameKey),
    ]);
    const viewerDisplayNames = Object.fromEntries(viewerDisplayNameEntries);

    return (
      <div className={classes(commonCss.page, padding(20, 'lrt'))}>
        {/* Overview section */}
        <CollapseButton
          sectionName={t(overviewSectionName)}
          collapseSections={collapseSections}
          compareSetState={this.setStateSafe.bind(this)}
        />
        {!collapseSections[t(overviewSectionName)] && (
          <div className={commonCss.noShrink}>
            <RunList
              onError={this.showPageError.bind(this)}
              {...this.props}
              selectedIds={selectedIds}
              runIdListMask={runIds}
              disablePaging={true}
              onSelectionChange={this._selectionChanged.bind(this)}
            />
          </div>
        )}

        <Separator orientation='vertical' />

        {/* Parameters section */}
        <CollapseButton
          sectionName={t(paramsSectionName)}
          collapseSections={collapseSections}
          compareSetState={this.setStateSafe.bind(this)}
        />
        {!collapseSections[t(paramsSectionName)] && (
          <div className={classes(commonCss.noShrink, css.outputsRow)}>
            <Separator orientation='vertical' />
            <CompareTable {...this.state.paramsCompareProps} />
            <Hr />
          </div>
        )}

        {/* Metrics section */}
        <CollapseButton
          sectionName={t(metricsSectionName)}
          collapseSections={collapseSections}
          compareSetState={this.setStateSafe.bind(this)}
        />
        {!collapseSections[t(metricsSectionName)] && (
          <div className={classes(commonCss.noShrink, css.outputsRow)}>
            <Separator orientation='vertical' />
            <CompareTable {...this.state.metricsCompareProps} />
            <Hr />
          </div>
        )}

        <Separator orientation='vertical' />

        {Array.from(viewersMap.keys()).map(
          (viewerType, i) =>
            !!runsPerViewerType(viewerType).length && (
              <div key={i}>
                <CollapseButton
                  collapseSections={collapseSections}
                  compareSetState={this.setStateSafe.bind(this)}
                  sectionName={viewerDisplayNames[viewerType]}
                />
                {!collapseSections[viewerDisplayNames[viewerType]] && (
                  <React.Fragment>
                    <div className={classes(commonCss.flex, css.outputsRow)}>
                      {/* If the component allows aggregation, add one more card for
                its aggregated view. Only do this if there is more than one
                output, filtering out any unselected runs. */}
                      {componentMap[viewerType].isAggregatable &&
                        runsPerViewerType(viewerType).length > 1 && (
                          <PlotCard
                            configs={runsPerViewerType(viewerType).map(t => t.config)}
                            maxDimension={400}
                            title={t('aggregatedView')}
                          />
                        )}

                      {runsPerViewerType(viewerType).map((taggedConfig, c) => (
                        <PlotCard
                          key={c}
                          configs={[taggedConfig.config]}
                          title={taggedConfig.runName}
                          maxDimension={400}
                        />
                      ))}
                      <Separator />
                    </div>
                    <Hr />
                  </React.Fragment>
                )}
                <Separator orientation='vertical' />
              </div>
            ),
        )}
      </div>
    );
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public async load(): Promise<void> {
    this.clearBanner();

    const queryParamRunIds = new URLParser(this.props).get(QUERY_PARAMS.runlist);
    const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];
    const runs: ApiRunDetail[] = [];
    const workflowObjects: Workflow[] = [];
    const failingRuns: string[] = [];
    let lastError: Error | null = null;
    const { t } = this.props;

    await Promise.all(
      runIds.map(async id => {
        try {
          const run = await Apis.runServiceApi.getRun(id);
          runs.push(run);
          workflowObjects.push(JSON.parse(run.pipeline_runtime!.workflow_manifest || '{}'));
        } catch (err) {
          failingRuns.push(id);
          lastError = err;
        }
      }),
    );

    if (lastError) {
      await this.showPageError(
        `${t('errorLoadRuns1')} ${failingRuns.length}${t('errorLoadRuns2')}`,
        lastError,
      );
      logger.error(
        `Failed loading ${failingRuns.length} runs, last failed with the error: ${lastError}`,
      );
      return;
    }

    const selectedIds = runs.map(r => r.run!.id!);
    this.setStateSafe({
      runs,
      selectedIds,
      workflowObjects,
    });
    this._loadParameters(selectedIds);
    this._loadMetrics(selectedIds);

    const outputPathsList = workflowObjects.map(workflow =>
      WorkflowParser.loadAllOutputPaths(workflow),
    );

    // Maps a viewer type (ROC, table.. etc) to a list of all viewer instances
    // of that type, each tagged with its parent run id
    const viewersMap = new Map<PlotType, TaggedViewerConfig[]>();

    await Promise.all(
      outputPathsList.map(async (pathList, i) => {
        for (const path of pathList) {
          const configs = await OutputArtifactLoader.load(
            path,
            workflowObjects[0]?.metadata?.namespace,
          );
          configs.forEach(config => {
            const currentList: TaggedViewerConfig[] = viewersMap.get(config.type) || [];
            currentList.push({
              config,
              runId: runs[i].run!.id!,
              runName: runs[i].run!.name!,
            });
            viewersMap.set(config.type, currentList);
          });
        }
      }),
    );

    // For each output artifact type, list all artifact instances in all runs
    this.setStateSafe({ viewersMap });
  }

  protected _selectionChanged(selectedIds: string[]): void {
    this.setState({ selectedIds });
    this._loadParameters(selectedIds);
    this._loadMetrics(selectedIds);
  }

  private _collapseAllSections(): void {
    const { t } = this.props;
    const collapseSections = {
      [t(overviewSectionName)]: true,
      [t(paramsSectionName)]: true,
      [t(metricsSectionName)]: true,
    };
    Array.from(this.state.viewersMap.keys()).forEach(tz => {
      const sectionName = t(componentMap[tz].displayNameKey);
      collapseSections[sectionName] = true;
    });
    this.setState({ collapseSections });
  }

  private _loadParameters(selectedIds: string[]): void {
    const { runs, workflowObjects } = this.state;

    const selectedIndices = selectedIds.map(id => runs.findIndex(r => r.run!.id === id));
    const filteredRuns = runs.filter((_, i) => selectedIndices.indexOf(i) > -1);
    const filteredWorkflows = workflowObjects.filter((_, i) => selectedIndices.indexOf(i) > -1);

    const paramsCompareProps = CompareUtils.getParamsCompareProps(filteredRuns, filteredWorkflows);

    this.setState({ paramsCompareProps });
  }

  private _loadMetrics(selectedIds: string[]): void {
    const { runs } = this.state;

    const selectedIndices = selectedIds.map(id => runs.findIndex(r => r.run!.id === id));
    const filteredRuns = runs.filter((_, i) => selectedIndices.indexOf(i) > -1).map(r => r.run!);

    const metricsCompareProps = CompareUtils.multiRunMetricsCompareProps(filteredRuns);

    this.setState({ metricsCompareProps });
  }
}

const EnhancedCompare: React.FC<PageProps> = props => {
  const namespaceChanged = useNamespaceChangeEvent();
  const { t } = useTranslation(['common', 'experiments']);
  if (namespaceChanged) {
    // Compare page compares two runs, when namespace changes, the runs don't
    // exist in the new namespace, so we should redirect to experiment list page.
    return <Redirect to={RoutePage.EXPERIMENTS} />;
  }
  return <Compare {...props} t={t} />;
};

export default EnhancedCompare;

export const TEST_ONLY = {
  Compare,
};
