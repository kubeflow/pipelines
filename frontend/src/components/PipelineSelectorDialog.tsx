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
import ResourceSelector from 'src/pages/ResourceSelector';
import { Apis, PipelineSortKeys } from 'src/lib/Apis';
import { Column } from './CustomTable';
import { V2beta1Pipeline } from 'src/apisv2beta1/pipeline';
import Buttons from 'src/lib/Buttons';
import { PageProps } from 'src/pages/Page';
import { InspectionTabs } from './inspection/InspectionTabs';
import type { ToolbarActionMap } from 'src/lib/PageChromeTypes';
import { Toolbar } from './shell/PageChrome';
import { PipelineTabsHeaders, PipelineTabsTooltips } from 'src/pages/PipelinesPage';
import { BuildInfoContext } from 'src/lib/BuildInfo';
import { convertPipelineToResource } from 'src/lib/ResourceConverter';

import { Button } from './ui/button';
import { ModalDialog } from './ui/dialog';
import { errorToMessage } from 'src/lib/Utils';

enum NamespacedAndSharedTab {
  NAMESPACED = 0,
  SHARED = 1,
}

export interface PipelineSelectorDialogProps extends PageProps {
  open: boolean;
  onClose: (confirmed: boolean, selectedPipeline?: V2beta1Pipeline) => void;
  namespace: string | undefined; // use context or make it optional?
  pipelineSelectorColumns: Column[];
  toolbarActionMap?: ToolbarActionMap;
}

function PipelineSelectorDialog(props: PipelineSelectorDialogProps) {
  return <PipelineSelectionSession key={props.namespace || ''} {...props} />;
}

function PipelineSelectionSession(props: PipelineSelectorDialogProps) {
  const selectionGeneration = React.useRef(0);
  // External sync: invalidate pending selection reads when this namespace session unmounts.
  React.useEffect(
    () => () => {
      selectionGeneration.current++;
    },
    [],
  );
  const buildInfo = React.useContext(BuildInfoContext);
  const [view, setView] = React.useState(NamespacedAndSharedTab.NAMESPACED);
  const [unconfirmedSelectedPipeline, setUnconfirmedSelectedPipeline] =
    React.useState<V2beta1Pipeline>();

  function getPipelinesList(): React.JSX.Element {
    return (
      <ResourceSelector
        {...props}
        key={view}
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
            resources: response.pipelines?.map((p) => convertPipelineToResource(p)) || [],
          };
        }}
        columns={props.pipelineSelectorColumns}
        emptyMessage='No pipelines found. Upload a pipeline and then try again.'
        initialSortColumn={PipelineSortKeys.CREATED_AT}
        selectionChanged={async (selectedId: string) => {
          const generation = ++selectionGeneration.current;
          setUnconfirmedSelectedPipeline(undefined);
          try {
            const selectedPipeline = await Apis.pipelineServiceApiV2.getPipeline(selectedId);
            if (generation === selectionGeneration.current)
              setUnconfirmedSelectedPipeline(selectedPipeline);
          } catch (error) {
            const message = await errorToMessage(error);
            if (generation === selectionGeneration.current)
              props.updateDialog({
                title: 'Unable to select pipeline',
                content: message,
                buttons: [{ text: 'Dismiss' }],
              });
          }
        }}
      />
    );
  }

  function getTabs(): React.JSX.Element | null {
    if (!buildInfo?.apiServerMultiUser) {
      return getPipelinesList();
    }

    return (
      <InspectionTabs
        tabs={[
          {
            label: PipelineTabsHeaders.PRIVATE,
            tooltip: PipelineTabsTooltips.PRIVATE,
          },
          {
            label: PipelineTabsHeaders.SHARED,
            tooltip: PipelineTabsTooltips.SHARED,
          },
        ]}
        selectedTab={view}
        onSwitch={tabSwitched}
        ariaLabel='Pipeline scope'
      >
        {getPipelinesList()}
      </InspectionTabs>
    );
  }

  function tabSwitched(newTab: NamespacedAndSharedTab): void {
    selectionGeneration.current++;
    setUnconfirmedSelectedPipeline(undefined);
    setView(newTab);
  }

  function closeAndResetState(): void {
    selectionGeneration.current++;
    props.onClose(false);
    setUnconfirmedSelectedPipeline(undefined);
    setView(NamespacedAndSharedTab.NAMESPACED);
  }

  const getToolbar = (): React.JSX.Element => {
    let actions = new Buttons(props, () => {}).getToolbarActionMap();
    if (props.toolbarActionMap) {
      actions = props.toolbarActionMap;
    }
    return <Toolbar actions={actions} breadcrumbs={[]} pageTitle='' topLevelToolbar={false} />;
  };

  if (!props.open) return null;

  return (
    <ModalDialog
      open={props.open}
      id='pipelineSelectorDialog'
      size='lg'
      title='Choose a pipeline'
      onClose={closeAndResetState}
      actions={
        <>
          <Button id='cancelPipelineSelectionBtn' variant='secondary' onClick={closeAndResetState}>
            Cancel
          </Button>
          <Button
            id='usePipelineBtn'
            onClick={() => {
              selectionGeneration.current++;
              props.onClose(true, unconfirmedSelectedPipeline);
              setUnconfirmedSelectedPipeline(undefined);
              setView(NamespacedAndSharedTab.NAMESPACED);
            }}
            disabled={!unconfirmedSelectedPipeline}
          >
            Use this pipeline
          </Button>
        </>
      }
    >
      {getToolbar()}
      {getTabs()}
    </ModalDialog>
  );
}

export default PipelineSelectorDialog;
