import Template from '../lib/template';
import { Run } from '../lib/run';

export class TemplateClickEvent extends MouseEvent {
  public model: {
    template: Template,
  };
}

export class RunClickEvent extends MouseEvent {
  public model: {
    run: Run,
  };
}

export const ROUTE_EVENT = 'route';
export class RouteEvent extends CustomEvent {
  public detail: {
    path: string,
  }
  constructor(path: string) {
    const eventInit = {
      bubbles: true,
      detail: { path }
    };
    Object.defineProperty(eventInit, 'composed', {
      value: true,
      writable: false,
    });

    super(ROUTE_EVENT, eventInit);
  }
}

export class TabSelectedEvent extends CustomEvent {
  public detail: {
    value: number;
  }
}
