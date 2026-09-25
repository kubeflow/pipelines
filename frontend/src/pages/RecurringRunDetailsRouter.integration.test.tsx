/*
 * Copyright 2026 The Kubeflow Authors
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

import { render, screen, waitFor } from '@testing-library/react';
import { V2beta1RecurringRunStatus } from 'src/apisv2beta1/recurringrun';
import { RouteParams } from 'src/components/Router';
import * as features from 'src/features';
import { Apis } from 'src/lib/Apis';
import TestUtils from 'src/TestUtils';
import { CommonTestWrapper } from 'src/TestWrapper';
import RecurringRunDetailsRouter from './RecurringRunDetailsRouter';
import RecurringRunDetailsV2 from './RecurringRunDetailsV2';

const recurringRunId = 'latest-recurring-run';

afterEach(() => vi.restoreAllMocks());

describe.each([false, true])('native recurring-run details (functional=%s)', (functional) => {
  it.each([undefined, 'deleted-version'])(
    'keeps metadata and actions available without loading pipeline version %s',
    async (pipelineVersionId) => {
      vi.spyOn(features, 'isFeatureEnabled').mockImplementation(
        (key) => key === features.FeatureKey.FUNCTIONAL_COMPONENT && functional,
      );
      vi.spyOn(Apis.recurringRunServiceApi, 'getRecurringRun').mockResolvedValue({
        recurring_run_id: recurringRunId,
        display_name: 'Recurring training',
        description: 'Runs the latest pipeline version',
        pipeline_version_reference: {
          pipeline_id: 'training-pipeline',
          pipeline_version_id: pipelineVersionId,
        },
        runtime_config: { parameters: { dataset: 'training-data' } },
        status: V2beta1RecurringRunStatus.ENABLED,
        trigger: { periodic_schedule: { interval_second: '3600' } },
      });
      const getPipelineVersion = vi
        .spyOn(Apis.pipelineServiceApiV2, 'getPipelineVersion')
        .mockRejectedValue(new Error('Pipeline version no longer exists'));
      const updateToolbar = vi.fn();
      const props = TestUtils.generatePageProps(
        RecurringRunDetailsV2,
        '' as any,
        { [RouteParams.recurringRunId]: recurringRunId },
        vi.fn(),
        vi.fn(),
        vi.fn(),
        updateToolbar,
        vi.fn(),
      );

      render(
        <CommonTestWrapper>
          <RecurringRunDetailsRouter {...props} />
        </CommonTestWrapper>,
      );

      expect(await screen.findByText('training-data')).toBeInTheDocument();
      expect(screen.getByText('Runs the latest pipeline version')).toBeInTheDocument();
      await waitFor(() =>
        expect(updateToolbar).toHaveBeenCalledWith(
          expect.objectContaining({
            pageTitle: 'Recurring training',
            actions: expect.objectContaining({
              cloneRecurringRun: expect.objectContaining({ title: 'Clone recurring run' }),
              refresh: expect.objectContaining({ title: 'Refresh' }),
              enableRecurringRun: expect.objectContaining({ title: 'Enable', disabled: true }),
              disableRecurringRun: expect.objectContaining({ title: 'Disable', disabled: false }),
              deleteRun: expect.objectContaining({ title: 'Delete' }),
            }),
          }),
        ),
      );
      expect(getPipelineVersion).not.toHaveBeenCalled();
      expect(screen.queryByRole('alert')).toBeNull();
    },
  );
});
