export enum Deployments {
  KUBEFLOW = 'KUBEFLOW',
  MARKETPLACE = 'MARKETPLACE',
}

const DEPLOYMENT_DEFAULT = undefined;
// Uncomment this to debug marketplace:
// const DEPLOYMENT_DEFAULT = Deployments.MARKETPLACE;

export const KFP_FLAGS = {
  DEPLOYMENT:
    // tslint:disable-next-line:no-string-literal
    window && window['KFP_FLAGS']
      ? // tslint:disable-next-line:no-string-literal
        window['KFP_FLAGS']['DEPLOYMENT'] === Deployments.KUBEFLOW
        ? Deployments.KUBEFLOW
        : // tslint:disable-next-line:no-string-literal
        window['KFP_FLAGS']['DEPLOYMENT'] === Deployments.MARKETPLACE
        ? Deployments.MARKETPLACE
        : DEPLOYMENT_DEFAULT
      : DEPLOYMENT_DEFAULT,
};
