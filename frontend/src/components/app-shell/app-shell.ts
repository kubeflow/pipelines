// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import 'app-route/app-location.html';
import 'app-route/app-route.html';
import 'iron-icons/iron-icons.html';
import 'iron-pages/iron-pages.html';
import 'paper-progress/paper-progress.html';
import 'paper-styles/paper-styles.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';
import { PageError } from '../page-error/page-error';

import '../page-error/page-error';
import '../side-nav/side-nav';

import './app-shell.html';

@customElement('app-shell')
export class AppShell extends Polymer.Element {

  @property({ type: String })
  public page = '';

  @property({ type: Object })
  public route?: {
    prefix: string,
    path: string,
    __queryParams: { [k: string]: string },
    __data: any,
  } = undefined;

  private _debouncer: Polymer.Debouncer | undefined = undefined;

  private _ROUTES: { [route: string]: string } = {
    '/': 'pipeline-list',
    '/jobRun': 'run-details',
    '/jobs': 'job-list',
    '/jobs/details': 'job-details',
    '/jobs/new': 'job-new',
    '/pipelines': 'pipeline-list',
  };

  static get observers(): string[] {
    return ['_routePathChanged(route.path)'];
  }

  public get errorElement(): PageError {
    return this.$.errorEl as PageError;
  }

  public ready(): void {
    super.ready();
    this.addEventListener(RouteEvent.name, this._routeEventListener.bind(this));
  }

  protected async _routePathChanged(newPath: string): Promise<void> {

    // Workaround for https://github.com/PolymerElements/app-route/issues/173
    // to handle navigation events only once.
    this._debouncer = Polymer.Debouncer.debounce(
        this._debouncer || null,
        Polymer.Async.timeOut.after(100),
        async () => {

          // TODO: Add exponential backoff
          while (!await Apis.isApiServerReady()) {
            this.errorElement.error = 'Could not reach the backend server. Retrying..';
            await Utils.wait(2000);
          }
          this.errorElement.error = '';

          const parts = newPath.substr(1).split('/');
          const args = parts.splice(2).join('/');
          let pageName = `/${parts.join('/')}`;
          if (pageName === '/') {
            pageName = '/pipelines';
          }

          const elementName = this._ROUTES[pageName];
          if (!elementName) {
            this.errorElement.error = 'Cannot find page: ' + pageName;
            throw new Error('Cannot find page: ' + pageName);
          }
          await import (`../${elementName}/${elementName}`);
          const el = this.$.pages.querySelector(elementName) as PageElement;
          el.load(args, this.route!.__queryParams, this.route!.__data);
          this.page = pageName;
        }
    );
  }

  private _routeEventListener(e: RouteEvent): void {
    const url = new URL(e.detail.path, window.location.href);
    this.set('route.path', url.pathname);
    const queryParams = {} as any;
    for (const entry of url.searchParams.entries()) {
      queryParams[entry[0]] = entry[1];
    }
    this.set('route.__queryParams', queryParams);
    this.set('route.__data', e.detail.data);
  }
}
