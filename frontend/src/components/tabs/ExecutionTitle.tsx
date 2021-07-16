/*
 * Copyright 2021 The Kubeflow Authors
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

import { Execution } from 'src/third_party/mlmd';
import React from 'react';
import { Link } from 'react-router-dom';
import { commonCss } from 'src/Css';
import { ExecutionHelpers } from 'src/mlmd/MlmdUtils';
import { RoutePageFactory } from '../Router';

interface ExecutionTitleProps {
  execution: Execution;
}

export function ExecutionTitle({ execution }: ExecutionTitleProps) {
  return (
    <>
      <div>
        This step corresponds to execution{' '}
        <Link className={commonCss.link} to={RoutePageFactory.executionDetails(execution.getId())}>
          "{ExecutionHelpers.getName(execution)}".
        </Link>
      </div>
    </>
  );
}
