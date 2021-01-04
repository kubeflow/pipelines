/*
 * Copyright 2020 Arrikto Inc.
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
import Switch, { SwitchProps } from '@material-ui/core/Switch';
import FormGroup from '@material-ui/core/FormGroup';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Tooltip from '@material-ui/core/Tooltip';
import Card from '@material-ui/core/Card';
import CardContent from '@material-ui/core/CardContent';
import { withStyles } from '@material-ui/core/styles';
import { stylesheet } from 'typestyle';
import { color, fontsize } from '../Css';
import {ExternalLink} from "../atoms/ExternalLink";

const css = stylesheet({
  reduceGraphPosition: {
    position: 'absolute',
    top: 0,
    left: 0,
  },
});

const NostyleTooltip = withStyles({
  tooltip: {
    backgroundColor: 'transparent',
    border: '0 none',
    color: color.secondaryText,
    fontSize: fontsize.base,
    maxWidth: 220,
  },
})(Tooltip);

interface ReduceGraphSwitchProps extends SwitchProps {}

const ReduceGraphSwitch = (props: ReduceGraphSwitchProps) => {
  return (
    <div className={css.reduceGraphPosition}>
      <NostyleTooltip
        title={
          <Card>
            <CardContent>
                <div>
                  <p>
                    Enables a{' '}
                    <ExternalLink href='https://en.wikipedia.org/wiki/Transitive_reduction'>
                      transitive reduction
                    </ExternalLink>{' '}
                    of the pipeline graph, hiding all the redundant edges. This
                    option is just a visualization helper and does not have any
                    permanent effect on the pipeline itself.
                  </p>
                  <p>
                    Note that edges are removed regardless of their function in
                    the pipeline, so the reduced form might not provide enough
                    details to understand how the pipeline really behaves.
                  </p>
                </div>
            </CardContent>
          </Card>
        }
        interactive={true}
        leaveDelay={200}
        placement='top'
      >
        <FormGroup>
          <FormControlLabel
            control={<Switch color={'primary'} {...props} />}
            label={'Simplify Graph'}
            labelPlacement='end'
          />
        </FormGroup>
      </NostyleTooltip>
    </div>
  );
};

export default ReduceGraphSwitch;
