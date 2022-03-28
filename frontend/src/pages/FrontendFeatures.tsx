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

import { Button, Switch, TableCell } from '@material-ui/core';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import * as React from 'react';
import { commonCss, padding } from 'src/Css';
import { getFeatureList, initFeatures, saveFeatures } from 'src/features';
import { classes } from 'typestyle';

interface FrontendFeaturesProps {}

// Secret page which developer can enable/disable pre-release features on UI.
const FrontendFeatures: React.FC<FrontendFeaturesProps> = () => {
  initFeatures();
  const srcFeatures = getFeatureList();
  const [features, setFeatures] = React.useState(srcFeatures);

  const reset = () => {
    setFeatures(srcFeatures);
  };
  const submit = () => {
    saveFeatures(features);
    setFeatures(getFeatureList());
  };

  const toggleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const index = features.findIndex(f => f.name === event.target.name);
    if (index < 0) {
      console.log(`unable to find index for feature name: ${event.target.name}`);
      return;
    }
    const newFeatures = [...features];
    newFeatures[index].active = event.target.checked;
    setFeatures(newFeatures);
  };

  return (
    <div className={classes(commonCss.page, padding(20, 't'))}>
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div style={{ alignSelf: 'flex-end', flexDirection: 'initial' }}>
          <Button variant='contained' color='primary' onClick={submit}>
            Save changes
          </Button>
          <Button variant='contained' color='secondary' onClick={reset}>
            Reset
          </Button>
        </div>
        <Table aria-label=' table'>
          <TableHead>
            <TableRow>
              <TableCell>Feature Flag Name</TableCell>
              <TableCell align='left'>Description</TableCell>
              <TableCell align='right'>Enabled</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {features.map(f => (
              <TableRow key={f.name}>
                <TableCell component='th' scope='row'>
                  {f.name}
                </TableCell>
                <TableCell align='left'>{f.description}</TableCell>
                <TableCell align='right'>
                  <Switch
                    checked={f.active}
                    onChange={toggleChange}
                    color='primary'
                    name={f.name}
                    inputProps={{ 'aria-label': 'primary checkbox' }}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
};

export default FrontendFeatures;
