/// <reference path="../../../bower_components/polymer/types/polymer.d.ts" />
/// <reference path="../../../bower_components/polymer/types/lib/utils/templatize.d.ts" />
import * as Utils from '../../modules/utils';
import PageElement from '../../modules/page_element';
import { ROUTE_EVENT, RouteEvent } from '../../modules/events';
import { customElement } from '../../decorators';

@customElement
export default class PipelinesDashboard extends Polymer.Element {

  public page = '';
  public route: object | null = null;

  static get properties() {
    return {
      page: { type: String },
      route: { type: Object },
    };
  }

  static get observers() {
    return [ '_routePathChanged(route.path)' ];
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
        // the page name is the first two, to allow for things like templates/view
        // and job/view. The rest are the argument to that page.
        const args = parts.splice(2);
        this.page = `${parts.join('')}`;
        const pageEl = this._getPageElement(this.page);
        pageEl.refresh(args.join('/'));
      } else {
        Utils.log.error(`Bad path: ${newPath}`)
      }
    }
  }

  private _routeEventListener(e: RouteEvent) {
    this.set('route.path', e.detail.path);
  }

  private _getPageElement(pageName: string): PageElement {
    const el = this.$.pages.querySelector('[name=' + pageName + ']');
    if (!el) {
      throw new Error(`Cannot find page element: ${pageName}`);
    } else {
      // Temporary workaround for https://github.com/Polymer/polymer/issues/5074
      return (el as any);
    }
  }
}
