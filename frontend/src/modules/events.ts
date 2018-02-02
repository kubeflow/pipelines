import Template from '../modules/template';

export class DomRepeatMouseEvent extends MouseEvent {
  public model: {
    template: Template,
  };
}

export const ROUTE_EVENT = 'route';
export class RouteEvent extends CustomEvent<any> {
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
