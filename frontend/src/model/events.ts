// TODO: consider moving these into the event classes as static fields.
export enum EventName {
  LIST_FORMAT_CHANGE = 'listFormatChange',
  NEW_LIST_PAGE = 'newListPage',
  ROUTE = 'route',
}

export class ItemClickEvent extends CustomEvent {
  public detail: {
    index: number,
  };
}

export class ListFormatChangeEvent extends CustomEvent {
  public detail: {
    filterString: string,
    orderAscending: boolean,
    sortColumn: string,
  };
}

export class NewListPageEvent extends CustomEvent {
  public detail: {
    filterBy: string,
    pageNumber: number,
    pageToken: string,
    sortBy: string,
  };
}

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

    super(EventName.ROUTE, eventInit);
  }
}
