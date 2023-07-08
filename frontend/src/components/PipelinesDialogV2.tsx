/*
 * Copyright 2023 The Kubeflow Authors
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
import Button from '@material-ui/core/Button';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import { classes } from 'typestyle';
import { padding, commonCss } from 'src/Css';
import DialogContent from '@material-ui/core/DialogContent';
import ResourceSelector from 'src/pages/ResourceSelector';
import { Apis, PipelineSortKeys } from 'src/lib/Apis';
import { Column } from './CustomTable';
import { V2beta1Pipeline } from 'src/apisv2beta1/pipeline';
import Buttons from 'src/lib/Buttons';
import { PageProps } from 'src/pages/Page';
import MD2Tabs from 'src/atoms/MD2Tabs';
import Toolbar, { ToolbarActionMap } from 'src/components/Toolbar';
import { PipelineTabsHeaders, PipelineTabsTooltips } from 'src/pages/PrivateAndSharedPipelines';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import { convertPipelineToResource } from 'src/lib/ResourceConverter';

enum NamespacedAndSharedTab {
  NAMESPACED = 0,
  SHARED = 1,
}

export interface PipelinesDialogV2Props extends PageProps {
  open: boolean;
  selectorDialog: string;
  onClose: (confirmed: boolean, selectedPipeline?: V2beta1Pipeline) => void;
  namespace: string | undefined; // use context or make it optional?
  pipelineSelectorColumns: Column[];
  toolbarActionMap?: ToolbarActionMap;
}

const PipelinesDialogV2: React.FC<PipelinesDialogV2Props> = (props): JSX.Element | null => {
  const buildInfo = React.useContext(BuildInfoContext);
  const [view, setView] = React.useState(NamespacedAndSharedTab.NAMESPACED);
  const [unconfirmedSelectedPipeline, setUnconfirmedSelectedPipeline] = React.useState<
    V2beta1Pipeline
  >();

  function getPipelinesList(): JSX.Element {
    return (
      <ResourceSelector
        {...props}
        filterLabel='Filter pipelines'
        listApi={async (
          page_token?: string,
          page_size?: number,
          sort_by?: string,
          filter?: string,
        ) => {
          const response = await Apis.pipelineServiceApiV2.listPipelines(
            buildInfo?.apiServerMultiUser && view === NamespacedAndSharedTab.NAMESPACED
              ? props.namespace
              : undefined,
            page_token,
            page_size,
            sort_by,
            filter,
          );
          return {
            nextPageToken: response.next_page_token || '',
            resources: response.pipelines?.map(p => convertPipelineToResource(p)) || [],
          };
        }}
        columns={props.pipelineSelectorColumns}
        emptyMessage='No pipelines found. Upload a pipeline and then try again.'
        initialSortColumn={PipelineSortKeys.CREATED_AT}
        selectionChanged={async (selectedId: string) => {
          const selectedPipeline = await Apis.pipelineServiceApiV2.getPipeline(selectedId);
          setUnconfirmedSelectedPipeline(selectedPipeline);
        }}
      />
    );
  }

  function getTabs(): JSX.Element | null {
    if (!buildInfo?.apiServerMultiUser) {
      return null;
    }

    return (
      <MD2Tabs
        tabs={[
          {
            header: PipelineTabsHeaders.PRIVATE,
            tooltip: PipelineTabsTooltips.PRIVATE,
          },
          {
            header: PipelineTabsHeaders.SHARED,
            tooltip: PipelineTabsTooltips.SHARED,
          },
        ]}
        selectedTab={view}
        onSwitch={tabSwitched}
      />
    );
  }

  function tabSwitched(newTab: NamespacedAndSharedTab): void {
    setUnconfirmedSelectedPipeline(undefined);
    setView(newTab);
  }

  function closeAndResetState(): void {
    props.onClose(false);
    setUnconfirmedSelectedPipeline(undefined);
    setView(NamespacedAndSharedTab.NAMESPACED);
  }

  const getToolbar = (): JSX.Element => {
    let actions = new Buttons(props, () => {}).getToolbarActionMap();
    if (props.toolbarActionMap) {
      actions = props.toolbarActionMap;
    }
    return <Toolbar actions={actions} breadcrumbs={[]} pageTitle={'Choose a pipeline'} />;
  };

  return (
    <Dialog
      open={props.open}
      classes={{ paper: props.selectorDialog }}
      onClose={() => closeAndResetState()}
      PaperProps={{ id: 'pipelineSelectorDialog' }}
    >
      <DialogContent>
        {getToolbar()}
        <div className={classes(commonCss.page, padding(20, 't'))}>
          {getTabs()}

          {view === NamespacedAndSharedTab.NAMESPACED && getPipelinesList()}
          {view === NamespacedAndSharedTab.SHARED && getPipelinesList()}
        </div>
      </DialogContent>
      <DialogActions>
        <Button
          id='cancelPipelineSelectionBtn'
          onClick={() => closeAndResetState()}
          color='secondary'
        >
          Cancel
        </Button>
        <Button
          id='usePipelineBtn'
          onClick={() => props.onClose(true, unconfirmedSelectedPipeline)}
          color='secondary'
          disabled={!unconfirmedSelectedPipeline}
        >
          Use this pipeline
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default PipelinesDialogV2;
