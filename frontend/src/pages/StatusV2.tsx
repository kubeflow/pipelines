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
import ErrorIcon from '@material-ui/icons/Error';
import PendingIcon from '@material-ui/icons/Schedule';
import RunningIcon from 'src/icons/statusRunning';
import SkippedIcon from '@material-ui/icons/SkipNext';
import SuccessIcon from '@material-ui/icons/CheckCircle';
import CachedIcon from 'src/icons/statusCached';
import TerminatedIcon from 'src/icons/statusTerminated';
import Tooltip from '@material-ui/core/Tooltip';
import UnknownIcon from '@material-ui/icons/Help';
import { color } from 'src/Css';
import { logger, formatDateString } from 'src/lib/Utils';
import { checkIfTerminatedV2 } from 'src/lib/StatusUtils';
import { V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { Execution } from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_pb';

export function statusToIcon(
  state?: V2beta1RuntimeState,
  startDate?: Date | string,
  endDate?: Date | string,
  nodeMessage?: string,
  mlmdState?: Execution.State,
): JSX.Element {
  state = checkIfTerminatedV2(state, nodeMessage);
  // tslint:disable-next-line:variable-name
  let IconComponent: any = UnknownIcon;
  let iconColor = color.inactive;
  let title = 'Unknown status';
  switch (state) {
    case V2beta1RuntimeState.FAILED:
      IconComponent = ErrorIcon;
      iconColor = color.errorText;
      title = 'Resource failed to execute';
      break;
    case V2beta1RuntimeState.PENDING:
      IconComponent = PendingIcon;
      iconColor = color.weak;
      title = 'Pending execution';
      break;
    case V2beta1RuntimeState.RUNNING:
      IconComponent = RunningIcon;
      iconColor = color.blue;
      title = 'Running';
      break;
    case V2beta1RuntimeState.CANCELING:
      IconComponent = RunningIcon;
      iconColor = color.blue;
      title = 'Run is canceling';
      break;
    case V2beta1RuntimeState.SKIPPED:
      IconComponent = SkippedIcon;
      title = 'Execution has been skipped for this resource';
      break;
    case V2beta1RuntimeState.SUCCEEDED:
      IconComponent = SuccessIcon;
      iconColor = color.success;
      title = 'Executed successfully';
      break;
    case V2beta1RuntimeState.CANCELED:
      IconComponent = TerminatedIcon;
      iconColor = color.terminated;
      title = 'Run was manually canceled';
      break;
    case V2beta1RuntimeState.RUNTIMESTATEUNSPECIFIED:
      break;
    default:
      logger.verbose('Unknown state:', state);
  }
  // TODO(jlyaoyuli): Additional changes is probably needed after Status IR integration.
  if (mlmdState === Execution.State.CACHED) {
    IconComponent = CachedIcon;
    iconColor = color.success;
    title = 'Execution was skipped and outputs were taken from cache';
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
