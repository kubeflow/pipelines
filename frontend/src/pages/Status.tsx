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
import { logger } from '../lib/Utils';

export enum NodePhase {
  ERROR = 'Error',
  FAILED = 'Failed',
  PENDING = 'Pending',
  RUNNING = 'Running',
  SKIPPED = 'Skipped',
  SUCCEEDED = 'Succeeded',
  UNKNOWN = 'Unknown',
}

export function statusToIcon(status: NodePhase): JSX.Element {
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
      iconColor = color.success;
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

  return <Tooltip title={title}><span>
    <IconComponent style={{ color: iconColor, height: 18, width: 18 }} />
  </span></Tooltip>;
}
