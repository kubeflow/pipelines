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
import { Page } from './Page';
import { ToolbarProps } from '../components/Toolbar';
import { TFunction } from 'i18next';
import { withTranslation } from 'react-i18next';

class Page404 extends Page<{ t: TFunction }, {}> {
  public getInitialToolbarState(): ToolbarProps {
    const { t } = this.props;
    return { actions: {}, breadcrumbs: [], pageTitle: '', t };
  }

  public async refresh(): Promise<void> {
    return;
  }

  public render(): JSX.Element {
    const { t } = this.props;
    return (
      <div style={{ margin: '100px auto', textAlign: 'center' }}>
        <div style={{ color: '#aaa', fontSize: 50, fontWeight: 'bold' }}>404</div>
        <div style={{ fontSize: 16 }}>
          {t('pageNotFound')}: {this.props.location.pathname}
        </div>
      </div>
    );
  }
}

export default withTranslation('common')(Page404);
