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

import Markdown from 'markdown-to-jsx';
import * as React from 'react';
import { classes, cssRaw } from 'typestyle';
import { ApiFilter, PredicateOp } from '../apis/filter/api';
import { AutoLink } from '../atoms/ExternalLink';
import { RoutePageFactory } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import SAMPLE_CONFIG from '../config/sample_config_from_backend.json';
import { commonCss, padding } from '../Css';
import { Apis } from '../lib/Apis';
import Buttons from '../lib/Buttons';
import { Page } from './Page';

const DEMO_PIPELINES: string[] = SAMPLE_CONFIG.slice(0, 4);
const DEMO_PIPELINES_ID_MAP = {
  control: 4,
  data: 3,
  tfxKeras: 2,
  tfx: 1,
  xgboost: 0,
};

const PAGE_CONTENT_MD = ({
  control,
  data,
  tfxKeras,
  tfx,
  xgboost,
}: {
  control: string;
  data: string;
  tfxKeras: string;
  tfx: string;
  xgboost: string;
}) => `
<br/>

## Build your own pipeline with

  * TensorFlow Extended (TFX) [SDK](https://www.tensorflow.org/tfx/guide) with end-to-end ML Pipeline Template ([Open TF 2.1 Notebook](https://console.cloud.google.com/mlengine/notebooks/deploy-notebook?q=download_url%3Dhttps%253A%252F%252Fraw.githubusercontent.com%252Ftensorflow%252Ftfx%252Fmaster%252Fdocs%252Ftutorials%252Ftfx%252Ftemplate.ipynb))
  * Kubeflow Pipelines [SDK](https://www.kubeflow.org/docs/pipelines/sdk/)

<br/>

## Demonstrations and Tutorials
This section contains demo and tutorial pipelines.

**Demos** - Try an end-to-end demonstration pipeline.

  * [TFX pipeline demo with Keras](${tfxKeras}) - Classification pipeline based on Keras. [source code](https://github.com/kubeflow/pipelines/tree/master/samples/core/iris)
  * [TFX pipeline demo with Estimator](${tfx}) - Classification pipeline with model analysis, based on a public BigQuery dataset of taxicab trips. [source code](https://github.com/kubeflow/pipelines/tree/master/samples/core/parameterized_tfx_oss)
  * [XGBoost Pipeline demo](${xgboost}) - An example of end-to-end distributed training for an XGBoost model. [source code](https://github.com/kubeflow/pipelines/tree/master/samples/core/xgboost_training_cm)

<br/>

**Tutorials** - Learn pipeline concepts by following a tutorial.

  * [Data passing in python components](${data}) - Shows how to pass data between python components. [source code](https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/Data%20passing%20in%20python%20components)
  * [DSL - Control structures](${control}) - Shows how to use conditional execution and exit handlers. [source code](https://github.com/kubeflow/pipelines/tree/master/samples/tutorials/DSL%20-%20Control%20structures)

Want to learn more? [Learn from sample and tutorial pipelines.](https://www.kubeflow.org/docs/pipelines/tutorials/)
`;

cssRaw(`
.kfp-start-page li {
  font-size: 14px;
  margin-block-start: 0.83em;
  margin-block-end: 0.83em;
  margin-left: 2em;
}
.kfp-start-page p {
  font-size: 14px;
  margin-block-start: 0.83em;
  margin-block-end: 0.83em;
}
.kfp-start-page h2 {
  font-size: 18px;
  margin-block-start: 1em;
  margin-block-end: 1em;
}
.kfp-start-page h3 {
  font-size: 16px;
  margin-block-start: 1em;
  margin-block-end: 1em;
}
`);

const OPTIONS = {
  overrides: { a: { component: AutoLink } },
};

export class GettingStarted extends Page<{}, { links: string[] }> {
  public state = {
    links: ['', '', '', ''].map(getPipelineLink),
  };

  public getInitialToolbarState(): ToolbarProps {
    const buttons = new Buttons(this.props, this.refresh.bind(this));
    return {
      actions: buttons.getToolbarActionMap(),
      breadcrumbs: [],
      pageTitle: 'Getting Started',
    };
  }

  public async componentDidMount() {
    const ids = await Promise.all(
      DEMO_PIPELINES.map(name =>
        Apis.pipelineServiceApi
          .listPipelines(undefined, 10, undefined, createAndEncodeFilter(name))
          .then(pipelineList => {
            const pipelines = pipelineList.pipelines;
            if (pipelines?.length !== 1) {
              // This should be accurate, do not accept ambiguous results.
              return '';
            }
            return pipelines[0].id || '';
          })
          .catch(() => ''),
      ),
    );
    this.setState({ links: ids.map(getPipelineLink) });
  }

  public async refresh() {
    this.componentDidMount();
  }

  public render(): JSX.Element {
    return (
      <div className={classes(commonCss.page, padding(20, 'lr'), 'kfp-start-page')}>
        <Markdown options={OPTIONS}>
          {PAGE_CONTENT_MD({
            control: this.state.links[DEMO_PIPELINES_ID_MAP.control],
            data: this.state.links[DEMO_PIPELINES_ID_MAP.data],
            tfxKeras: this.state.links[DEMO_PIPELINES_ID_MAP.tfxKeras],
            tfx: this.state.links[DEMO_PIPELINES_ID_MAP.tfx],
            xgboost: this.state.links[DEMO_PIPELINES_ID_MAP.xgboost],
          })}
        </Markdown>
      </div>
    );
  }
}

function getPipelineLink(id: string) {
  if (!id) {
    return '#/pipelines';
  }
  return `#${RoutePageFactory.pipelineDetails(id)}`;
}

function createAndEncodeFilter(filterString: string): string {
  const filter: ApiFilter = {
    predicates: [
      {
        key: 'name',
        op: PredicateOp.EQUALS,
        string_value: filterString,
      },
    ],
  };
  return encodeURIComponent(JSON.stringify(filter));
}
