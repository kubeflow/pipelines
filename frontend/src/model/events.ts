export class ItemClickEvent extends CustomEvent {
  public detail: {
    index: number,
  };
}

export const FILTER_CHANGED_EVENT = 'filterChanged';
export class FilterChangedEvent extends CustomEvent {
  public detail: {
    filterString: string,
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
