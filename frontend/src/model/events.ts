export class ItemClickEvent extends CustomEvent {
  public detail: {
    index: number,
  };
}

export const ROUTE_EVENT = 'route';
export class RouteEvent extends CustomEvent {
  public detail: {
    path: string,
    data?: {},
  };
  constructor(path: string, data?: {}) {
    const eventInit = {
      bubbles: true,
      detail: { path, data }
    };
    Object.defineProperty(eventInit, 'composed', {
      value: true,
      writable: false,
    });

    super(ROUTE_EVENT, eventInit);
  }
}
