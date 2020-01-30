/*
 * Copyright 2019 Google LLC
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
import Buttons from '../lib/Buttons';
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { classes } from 'typestyle';
import { commonCss, padding } from '../Css';

export class GettingStarted extends Page<{}, {}> {
  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons.getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: 'Getting Started',
    };
  }

  public async refresh() {
    // do nothing
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'))}>
        <div>
          <p className={classes(commonCss.header2)}>Getting Started - Build a pipeline</p>
          <p className={classes(commonCss.textField, padding(20, 'lr'))}>
            Classification
            <a
              className={classes(commonCss.link, padding(20, 'lr'))}
              href='https://console.cloud.google.com/mlengine/notebooks/deploy-notebook?q=download_url%3Dhttps%253A%252F%252Fraw.githubusercontent.com%252Fkubeflow%252Fpipelines%252F0.1.40%252Fsamples%252Fcore%252Fparameterized_tfx_oss%252Ftaxi_pipeline_notebook.ipynb'
            >
              Start Here
            </a>
          </p>
          <p>
            The table below provides a few demo and tutorial pipelines, and also allows you to
            upload your own pipelines. You can access additional samples and tutorials at
            <a
              className={classes(commonCss.link)}
              href='https://github.com/kubeflow/pipelines/tree/master/samples'
            >
              pipelines Github Repo.
            </a>
          </p>
        </div>
      </div>
    );
  }
}
