// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

abstract class BaseCustomEvent extends CustomEvent<any> {
  public abstract detail: any;

  constructor(eventType: string, detail: any) {
    super(eventType, {
      bubbles: true,
      composed: true,
      detail,
    });
  }
}

export class ItemDblClickEvent extends BaseCustomEvent {
  public detail!: {
    index: number,
  };

  constructor(index: number) {
    super(ItemDblClickEvent.name, { index } );
  }
}

export class ListFormatChangeEvent extends BaseCustomEvent {
  public detail!: {
    filterString: string,
    orderAscending: boolean,
    pageSize: number,
    sortColumn: string,
  };

  constructor(filterString: string, orderAscending: boolean, pageSize: number, sortColumn: string) {
    super(ListFormatChangeEvent.name, { filterString, orderAscending, pageSize, sortColumn });
  }
}

export class NewListPageEvent extends BaseCustomEvent {
  public detail!: {
    filterBy: string,
    pageNumber: number,
    pageSize: number,
    pageToken: string,
    sortBy: string,
  };

  constructor(
      filterBy: string,
      pageNumber: number,
      pageSize: number,
      pageToken: string,
      sortBy: string) {
    super(NewListPageEvent.name, { filterBy, pageNumber, pageSize, pageToken, sortBy });
  }
}

export class RouteEvent extends BaseCustomEvent {
  public detail!: {
    path: string,
    data?: {},
  };

  constructor(path: string, data?: {}) {
    super(RouteEvent.name, { path, data });
  }
}
