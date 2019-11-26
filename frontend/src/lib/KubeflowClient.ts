import { KFP_FLAGS, Deployments } from './Flags';
import { logger } from './Utils';

declare global {
  interface Window {
    // Provided by:
    // 1. https://github.com/kubeflow/kubeflow/tree/master/components/centraldashboard#client-side-library
    // 2. /frontend/server/server.ts -> KUBEFLOW_CLIENT_PLACEHOLDER
    centraldashboard: any;
  }
}

export function init(): void {
  if (KFP_FLAGS.DEPLOYMENT !== Deployments.KUBEFLOW) {
    return;
  }

  try {
    // Init method will invoke the callback with the event handler instance
    // and a boolean indicating whether the page is iframed or not
    window.centraldashboard.CentralDashboardEventHandler.init((cdeh: any) => {
      // Binds a callback that gets invoked anytime the Dashboard's
      // namespace is changed
      cdeh.onNamespaceSelected = (namespace: string) => {
        // tslint:disable-next-line:no-console
        console.log('Namespace changed to', namespace);
      };
    });
  } catch (err) {
    logger.error('Failed to initialize central dashboard client', err);
  }
}
