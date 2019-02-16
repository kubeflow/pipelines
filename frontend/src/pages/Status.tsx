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
import ErrorIcon from '@material-ui/icons/Error';
import PendingIcon from '@material-ui/icons/Schedule';
import RunningIcon from '../icons/statusRunning';
import SkippedIcon from '@material-ui/icons/SkipNext';
import SuccessIcon from '@material-ui/icons/CheckCircle';
import Tooltip from '@material-ui/core/Tooltip';
import UnknownIcon from '@material-ui/icons/Help';
import { color } from '../Css';
import { logger, formatDateString } from '../lib/Utils';

export enum NodePhase {
  ERROR = 'Error',
  FAILED = 'Failed',
  PENDING = 'Pending',
  RUNNING = 'Running',
  SKIPPED = 'Skipped',
  SUCCEEDED = 'Succeeded',
  UNKNOWN = 'Unknown',
}

export function hasFinished(status?: NodePhase): boolean {
  if (!status) {
    return false;
  }

  switch (status) {
    case NodePhase.SUCCEEDED: // Fall through
    case NodePhase.FAILED: // Fall through
    case NodePhase.ERROR: // Fall through
    case NodePhase.SKIPPED:
      return true;
    case NodePhase.PENDING: // Fall through
    case NodePhase.RUNNING: // Fall through
    case NodePhase.UNKNOWN:
      return false;
    default:
      return false;
  }
}

const fadedStatusColors = {
  error: '#fce8e6',
  notStarted: '#f7f7f7',
  running: '#e8f0fe',
  stopOrSkip: '#f1f3f4',
  succeeded: '#e6f4ea',
  warning: '#fef7f0',
};

export function statusToFadedColor(status?: NodePhase): string {
  if (!status) {
    return fadedStatusColors.notStarted;
  }

  switch (status) {
    case NodePhase.ERROR:
      return fadedStatusColors.error;
    case NodePhase.FAILED:
      return fadedStatusColors.error;
    case NodePhase.PENDING:
      return fadedStatusColors.notStarted;
    case NodePhase.RUNNING:
      return fadedStatusColors.running;
    case NodePhase.SKIPPED:
      return fadedStatusColors.stopOrSkip;
    case NodePhase.SUCCEEDED:
      return fadedStatusColors.succeeded;
    case NodePhase.UNKNOWN:
      // fall through
    default:
      logger.verbose('Unknown node phase:', status);
      return fadedStatusColors.notStarted;
  }
}

export function statusToIcon(status?: NodePhase, startDate?: Date | string, endDate?: Date | string): JSX.Element {
  // tslint:disable-next-line:variable-name
  let IconComponent: any = UnknownIcon;
  let iconColor = color.inactive;
  let title = 'Unknown status';
  switch (status) {
    case NodePhase.ERROR:
      IconComponent = ErrorIcon;
      iconColor = color.errorText;
      title = 'Error while running this resource';
      break;
    case NodePhase.FAILED:
      IconComponent = ErrorIcon;
      iconColor = color.errorText;
      title = 'Resource failed to execute';
      break;
    case NodePhase.PENDING:
      IconComponent = PendingIcon;
      iconColor = color.weak;
      title = 'Pending execution';
      break;
    case NodePhase.RUNNING:
      IconComponent = RunningIcon;
      iconColor = color.blue;
      title = 'Running';
      break;
    case NodePhase.SKIPPED:
      IconComponent = SkippedIcon;
      title = 'Execution has been skipped for this resource';
      break;
    case NodePhase.SUCCEEDED:
      IconComponent = SuccessIcon;
      iconColor = color.success;
      title = 'Executed successfully';
      break;
    case NodePhase.UNKNOWN:
      break;
    default:
      logger.verbose('Unknown node phase:', status);
  }

  return (
    <Tooltip title={
      <div>
        <div>{title}</div>
        {/* These dates may actually be strings, not a Dates due to a bug in swagger's handling of dates */}
        {startDate && (<div>Start: {formatDateString(startDate)}</div>)}
        {endDate && (<div>End: {formatDateString(endDate)}</div>)}
      </div>
    }>
      <span style={{ height: 18 }}>
        <IconComponent style={{ color: iconColor, height: 18, width: 18 }} />
      </span>
    </Tooltip>
  );
}
