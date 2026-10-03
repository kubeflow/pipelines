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

import type * as React from 'react';
import { Page } from './Page';
import { ToolbarProps } from 'src/lib/PageChromeTypes';
import { Button } from '../components/ui/button';
import './SecondaryPages.css';

export default class Page404 extends Page<{}, {}> {
  public getInitialToolbarState(): ToolbarProps {
    return { actions: {}, breadcrumbs: [], pageTitle: '' };
  }

  public async refresh(): Promise<void> {
    return;
  }

  public render(): React.JSX.Element {
    return (
      <section className='kfp-secondary-page kfp-not-found'>
        <p className='kfp-not-found-code' aria-hidden>
          404
        </p>
        <h1>Page not found</h1>
        <p>
          <code>{this.props.location.pathname}</code> is not a Pipelines page.
        </p>
        <Button onClick={() => this.props.navigate('/pipelines')}>Go to pipelines</Button>
      </section>
    );
  }
}
