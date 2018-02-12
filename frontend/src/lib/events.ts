import Template from 'src/lib/template';
import { Run } from 'src/lib/run';
import { Instance } from 'src/lib/instance';

export class TemplateClickEvent extends MouseEvent {
  public model: {
    template: Template,
  };
}

export class InstanceClickEvent extends MouseEvent {
  public model: {
    instance: Instance,
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
