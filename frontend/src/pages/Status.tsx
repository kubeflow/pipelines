/*
 * Copyright 2018 The Kubeflow Authors
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
import ErrorIcon from '@material-ui/icons/Error';
import PendingIcon from '@material-ui/icons/Schedule';
import RunningIcon from '../icons/statusRunning';
import SkippedIcon from '@material-ui/icons/SkipNext';
import SuccessIcon from '@material-ui/icons/CheckCircle';
import BlockIcon from '@material-ui/icons/Block';
import CachedIcon from '../icons/statusCached';
import TerminatedIcon from '../icons/statusTerminated';
import Tooltip from '@material-ui/core/Tooltip';
import UnknownIcon from '@material-ui/icons/Help';
import { color } from '../Css';
import { logger, formatDateString } from '../lib/Utils';
import { NodePhase, checkIfTerminated } from '../lib/StatusUtils';
import { Execution } from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_pb';
import { t } from 'i18next';

export function statusToIcon(
  status?: NodePhase,
  startDate?: Date | string,
  endDate?: Date | string,
  nodeMessage?: string,
  mlmdState?: Execution.State,
): JSX.Element {
  status = checkIfTerminated(status, nodeMessage);
  // tslint:disable-next-line:variable-name
  let IconComponent: any = UnknownIcon;
  let iconColor = color.inactive;
  let title = t('Status.unknownStatus');
  switch (status) {
    case NodePhase.ERROR:
      IconComponent = ErrorIcon;
      iconColor = color.errorText;
      title = t('Status.errorWhileRunningThisResource');
      break;
    case NodePhase.FAILED:
      IconComponent = ErrorIcon;
      iconColor = color.errorText;
      title = t('Status.resourceFailedToExecute');
      break;
    case NodePhase.PENDING:
      IconComponent = PendingIcon;
      iconColor = color.weak;
      title = t('Status.pendingExecution');
      break;
    case NodePhase.RUNNING:
      IconComponent = RunningIcon;
      iconColor = color.blue;
      title = t('Status.running');
      break;
    case NodePhase.TERMINATING:
      IconComponent = RunningIcon;
      iconColor = color.blue;
      title = t('Status.runIsTerminating');
      break;
    case NodePhase.SKIPPED:
      IconComponent = SkippedIcon;
      title = t('Status.executionHasBeenSkippedForThisResource');
      break;
    case NodePhase.SUCCEEDED:
      IconComponent = SuccessIcon;
      iconColor = color.success;
      title = t('Status.executedSuccessfully');
      break;
    case NodePhase.CACHED: // This is not argo native, only applies to node.
      IconComponent = CachedIcon;
      iconColor = color.success;
      title = t('Status.executionWasSkippedAndOutputsWereTakenFromCache');
      break;
    case NodePhase.TERMINATED:
      IconComponent = TerminatedIcon;
      iconColor = color.terminated;
      title = t('Status.runWasManuallyTerminated');
      break;
    case NodePhase.OMITTED:
      IconComponent = BlockIcon;
      title = t('Status.runWasOmittedBecauseThePreviousStepFailed');
      break;
    case NodePhase.UNKNOWN:
      break;
    default:
      logger.verbose('Unknown node phase:', status);
  }
  if (mlmdState === Execution.State.CACHED) {
    IconComponent = CachedIcon;
    iconColor = color.success;
    title = t('Status.executionWasSkippedAndOutputsWereTakenFromCache');
  }
  return (
    <Tooltip
      title={
        <div>
          <div>{title}</div>
          {/* These dates may actually be strings, not a Dates due to a bug in swagger's handling of dates */}
          {startDate && <div>Start: {formatDateString(startDate)}</div>}
          {endDate && <div>End: {formatDateString(endDate)}</div>}
        </div>
      }
    >
      <div>
        <IconComponent
          data-testid='node-status-sign'
          style={{ color: iconColor, height: 18, width: 18 }}
        />
      </div>
    </Tooltip>
  );
}
