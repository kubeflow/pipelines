abstract class BaseCustomEvent extends CustomEvent {
  public abstract detail: any;

  constructor(eventType: string, detail: any) {
    const eventInit = {
      bubbles: true,
      detail
    };
    Object.defineProperty(eventInit, 'composed', {
      value: true,
      writable: false,
    });

    super(eventType, eventInit);
  }
}

export class ItemDblClickEvent extends BaseCustomEvent {
  public detail: {
    index: number,
  };

  constructor(index: number) {
    super(ItemDblClickEvent.name, { index } );
  }
}

export class ListFormatChangeEvent extends BaseCustomEvent {
  public detail: {
    filterString: string,
    orderAscending: boolean,
    sortColumn: string,
  };

  constructor(filterString: string, orderAscending: boolean, sortColumn: string) {
    super(ListFormatChangeEvent.name, { filterString, orderAscending, sortColumn });
  }
}

export class NewListPageEvent extends BaseCustomEvent {
  public detail: {
    filterBy: string,
    pageNumber: number,
    pageToken: string,
    sortBy: string,
  };

  constructor(filterBy: string, pageNumber: number, pageToken: string, sortBy: string) {
    super(NewListPageEvent.name, { filterBy, pageNumber, pageToken, sortBy });
  }
}

export class RouteEvent extends BaseCustomEvent {
  public detail: {
    path: string,
    data?: {},
  };

  constructor(path: string, data?: {}) {
    super(RouteEvent.name, { path, data });
  }
}
