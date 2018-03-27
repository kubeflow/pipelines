import { PipelinePackage } from '../model/pipeline_package';

export class PackageClickEvent extends MouseEvent {
  public model: {
    package: PipelinePackage,
  };
}

export class ItemClickEvent extends CustomEvent {
  public detail: {
    index: number,
  };
}

export const ROUTE_EVENT = 'route';
export class RouteEvent extends CustomEvent {
  public detail: {
    path: string,
  };
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
