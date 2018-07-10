import 'app-route/app-location.html';
import 'app-route/app-route.html';
import 'iron-pages/iron-pages.html';
import 'paper-progress/paper-progress.html';
import 'paper-styles/paper-styles.html';
import 'polymer/polymer.html';

import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { RouteEvent } from '../../model/events';
import { PageElement } from '../../model/page_element';

import '../job-details/job-details';
import '../job-list/job-list';
import '../page-error/page-error';
import '../pipeline-details/pipeline-details';
import '../pipeline-list/pipeline-list';
import '../pipeline-new/pipeline-new';
import '../pipeline-schedule/pipeline-schedule';
import './app-shell.html';

const defaultPage = 'pipelines';

@customElement('app-shell')
export class AppShell extends Polymer.Element {

  @property({ type: String })
  public page = '';

  @property({ type: Object })
  public route: object | null = null;

  @property({ type: Boolean })
  protected _serverNotReady = false;

  private _debouncer: Polymer.Debouncer;

  static get observers(): string[] {
    return ['_routePathChanged(route.path)'];
  }

  public ready(): void {
    super.ready();
    this.addEventListener(RouteEvent.name, this._routeEventListener.bind(this));
  }

  protected async _routePathChanged(newPath: string): Promise<void> {

    // Workaround for https://github.com/PolymerElements/app-route/issues/173
    // to handle navigation events only once.
    this._debouncer = Polymer.Debouncer.debounce(
        this._debouncer,
        Polymer.Async.timeOut.after(100),
        async () => {

          // TODO: Add exponential backoff
          while (!await Apis.isApiServerReady()) {
            this._serverNotReady = true;
            await Utils.wait(2000);
          }
          this._serverNotReady = false;

          if (newPath !== undefined) {
            const parts = newPath.substr(1).split('/');
            if (parts.length) {
              // If there's only one part, that's the page name. If there's more,
              // the page name is the first two, to allow for things like pipelines/details
              // and job/details. The rest are the argument to that page.
              const args = parts.splice(2).join('/');
              let pageName = `${parts.join('/')}`;
              // For root '/', return the default page
              if (!pageName) {
                pageName = defaultPage;
              }
              const pageEl = this._getPageElement(pageName);
              pageEl.load(args, (this.route as any).__queryParams, (this.route as any).__data);
              this.page = pageName;
            } else {
              Utils.log.error(`Bad path: ${newPath}`);
            }
          }
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

  private _getPageElement(pageName: string): PageElement {
    const el = this.$.pages.querySelector(`[path="${pageName}"]`);
    if (!el) {
      throw new Error(`Cannot find page element: ${pageName}`);
    } else {
      // Temporary workaround for https://github.com/Polymer/polymer/issues/5074
      return (el as any);
    }
  }
}
