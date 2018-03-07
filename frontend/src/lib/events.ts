import { Job } from '../lib/job';
import { PipelinePackage } from '../lib/pipeline_package';

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

export class JobClickEvent extends MouseEvent {
  public model: {
    job: Job,
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

export class TabSelectedEvent extends CustomEvent {
  public detail: {
    value: number;
  };
}
