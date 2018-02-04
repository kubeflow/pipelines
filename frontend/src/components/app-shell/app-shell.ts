import 'polymer/polymer.html';

import 'app-route/app-location.html';
import 'app-route/app-route.html';
import 'iron-pages/iron-pages.html';
import 'paper-styles/paper-styles.html';

import '../instance-details/instance-details';
import '../instance-list/instance-list';
import '../instance-new/instance-new';
import '../template-details/template-details';
import '../template-list/template-list';
import * as Utils from '../../lib/utils';
import PageElement from '../../lib/page_element';
import { ROUTE_EVENT, RouteEvent } from '../../lib/events';
import { customElement, property } from '../../decorators';

import './app-shell.html';

@customElement
export default class AppShell extends Polymer.Element {

  @property({ type: String })
  public page = '';

  @property({ type: Object })
  public route: object | null = null;

  static get observers() {
    return ['_routePathChanged(route.path)'];
  }

  ready() {
    super.ready();
    this.addEventListener(ROUTE_EVENT, this._routeEventListener.bind(this));
  }

  protected _routePathChanged(newPath: string) {
    if (newPath !== undefined) {
      const parts = newPath.substr(1).split('/');
      if (parts.length) {
        // If there's only one part, that's the page name. If there's more,
        // the page name is the first two, to allow for things like templates/details
        // and job/details. The rest are the argument to that page.
        const args = parts.splice(2);
        let pageName = `${parts.join('/')}`;
        // For root '/', return the default page: templates
        if (!pageName) {
          pageName = 'templates';
        }
        const pageEl = this._getPageElement(pageName);
        pageEl.refresh(args.join('/'));
        this.page = pageName;
      } else {
        Utils.log.error(`Bad path: ${newPath}`)
      }
    }
  }

  private _routeEventListener(e: RouteEvent) {
    this.set('route.path', e.detail.path);
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
