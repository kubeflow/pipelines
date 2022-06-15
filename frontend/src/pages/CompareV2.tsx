/*
 * Copyright 2022 The Kubeflow Authors
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

import React, { useEffect, useRef, useState } from 'react';
import { useQuery } from 'react-query';
import { ApiRunDetail } from 'src/apis/run';
import Hr from 'src/atoms/Hr';
import Separator from 'src/atoms/Separator';
import CollapseButtonSingle from 'src/components/CollapseButtonSingle';
import { QUERY_PARAMS, RoutePage } from 'src/components/Router';
import { commonCss, padding } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import Buttons from 'src/lib/Buttons';
import { URLParser } from 'src/lib/URLParser';
import { errorToMessage } from 'src/lib/Utils';
import { classes, stylesheet } from 'typestyle';
import { PageProps } from './Page';
import RunList from './RunList';
import { METRICS_SECTION_NAME, OVERVIEW_SECTION_NAME, PARAMS_SECTION_NAME } from './Compare';

const css = stylesheet({
  outputsRow: {
    marginLeft: 15,
    overflowX: 'auto',
  },
});

function CompareV2(props: PageProps) {
  const { updateBanner, updateToolbar } = props;

  const runlistRef = useRef<RunList>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [isOverviewCollapsed, setIsOverviewCollapsed] = useState(false);
  const [isParamsCollapsed, setIsParamsCollapsed] = useState(false);
  const [isMetricsCollapsed, setIsMetricsCollapsed] = useState(false);

  const queryParamRunIds = new URLParser(props).get(QUERY_PARAMS.runlist);
  const runIds = (queryParamRunIds && queryParamRunIds.split(',')) || [];

  // Retrieves run details.
  const { isError, data, refetch } = useQuery<ApiRunDetail[], Error>(
    ['run_details', { ids: runIds }],
    () => Promise.all(runIds.map(async id => await Apis.runServiceApi.getRun(id))),
    {
      staleTime: Infinity,
      onError: async error => {
        const errorMessage = await errorToMessage(error);
        updateBanner({
          additionalInfo: errorMessage ? errorMessage : undefined,
          message: `Error: failed loading ${runIds.length} runs. Click Details for more information.`,
          mode: 'error',
        });
      },
      onSuccess: () => props.updateBanner({}),
    },
  );

  useEffect(() => {
    const refresh = async () => {
      if (runlistRef.current) {
        await runlistRef.current.refresh();
      }
      await refetch();
    };

    const buttons = new Buttons(props, refresh);
    updateToolbar({
      actions: buttons
        .expandSections(() => {
          setIsOverviewCollapsed(false);
          setIsParamsCollapsed(false);
          setIsMetricsCollapsed(false);
        })
        .collapseSections(() => {
          setIsOverviewCollapsed(true);
          setIsParamsCollapsed(true);
          setIsMetricsCollapsed(true);
        })
        .refresh(refresh)
        .getToolbarActionMap(),
      breadcrumbs: [{ displayName: 'Experiments', href: RoutePage.EXPERIMENTS }],
      pageTitle: 'Compare runs',
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [updateToolbar, refetch]);

  useEffect(() => {
    if (data) {
      setSelectedIds(data.map(r => r.run!.id!));
    }
  }, [data]);

  const showPageError = async (message: string, error: Error | undefined) => {
    const errorMessage = await errorToMessage(error);
    updateBanner({
      additionalInfo: errorMessage ? errorMessage : undefined,
      message: message + (errorMessage ? ' Click Details for more information.' : ''),
    });
  };

  const selectionChanged = (selectedIds: string[]): void => {
    setSelectedIds(selectedIds);
  };

  if (isError) {
    return <></>;
  }

  return (
    <div className={classes(commonCss.page, padding(20, 'lrt'))}>
      {/* Overview section */}
      <CollapseButtonSingle
        sectionName={OVERVIEW_SECTION_NAME}
        collapseSection={isOverviewCollapsed}
        collapseSectionUpdate={setIsOverviewCollapsed}
      />
      {!isOverviewCollapsed && (
        <div className={commonCss.noShrink}>
          <RunList
            onError={showPageError}
            {...props}
            selectedIds={selectedIds}
            ref={runlistRef}
            runIdListMask={runIds}
            disablePaging={true}
            onSelectionChange={selectionChanged}
          />
        </div>
      )}

      <Separator orientation='vertical' />

      {/* Parameters section */}
      <CollapseButtonSingle
        sectionName={PARAMS_SECTION_NAME}
        collapseSection={isParamsCollapsed}
        collapseSectionUpdate={setIsParamsCollapsed}
      />
      {!isParamsCollapsed && (
        <div className={classes(commonCss.noShrink, css.outputsRow)}>
          <Separator orientation='vertical' />
          <p>Parameter Section V2</p>
          <Hr />
        </div>
      )}

      {/* Metrics section */}
      <CollapseButtonSingle
        sectionName={METRICS_SECTION_NAME}
        collapseSection={isMetricsCollapsed}
        collapseSectionUpdate={setIsMetricsCollapsed}
      />
      {!isMetricsCollapsed && (
        <div className={classes(commonCss.noShrink, css.outputsRow)}>
          <Separator orientation='vertical' />
          <p>Metrics Section V2</p>
          <Hr />
        </div>
      )}

      <Separator orientation='vertical' />
    </div>
  );
}

export default CompareV2;
