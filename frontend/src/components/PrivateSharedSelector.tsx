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
import { classes } from 'typestyle';
import Radio from '@material-ui/core/Radio';
import Tooltip from '@material-ui/core/Tooltip';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import { commonCss, padding } from '../Css';
import { PipelineTabsHeaders } from '../pages/PrivateAndSharedPipelines';

export interface PrivateSharedSelectorProps {
  onChange: (isPrivate: boolean) => void;
}

export enum PipelineButtonTooltips {
  PRIVATE = 'Only people who have access to this namespace will be able to view and use this pipeline.',
  SHARED = 'Everyone in your organization will be able to view and use this pipeline.',
}

const PrivateSharedSelector: React.FC<PrivateSharedSelectorProps> = (props): JSX.Element | null => {
  const [namespacedPipeline, setNamespacedPipeline] = React.useState(true);

  React.useEffect(() => {
    props.onChange(namespacedPipeline);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [namespacedPipeline]);

  return (
    <React.Fragment>
      <div>Select if the new pipeline will be private or shared.</div>
      <div className={classes(commonCss.flex, padding(10, 'b'))}>
        <Tooltip title={PipelineButtonTooltips.PRIVATE} placement='top-start'>
          <FormControlLabel
            id='createNewPrivatePipelineBtn'
            label={PipelineTabsHeaders.PRIVATE}
            checked={namespacedPipeline === true}
            control={<Radio color='primary' />}
            onChange={() => {
              setNamespacedPipeline(true);
            }}
          />
        </Tooltip>
        <Tooltip title={PipelineButtonTooltips.SHARED} placement='top-start'>
          <FormControlLabel
            id='createNewSharedPipelineBtn'
            label={PipelineTabsHeaders.SHARED}
            checked={namespacedPipeline === false}
            control={<Radio color='primary' />}
            onChange={() => {
              setNamespacedPipeline(false);
            }}
          />
        </Tooltip>
      </div>
    </React.Fragment>
  );
};

export default PrivateSharedSelector;
