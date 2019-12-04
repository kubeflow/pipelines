export enum Deployments {
  KUBEFLOW = 'KUBEFLOW',
}

export const KFP_FLAGS = {
  DEPLOYMENT:
    // tslint:disable-next-line:no-string-literal
    window && window['KFP_FLAGS'] && window['KFP_FLAGS']['DEPLOYMENT'] === Deployments.KUBEFLOW
      ? Deployments.KUBEFLOW
      : undefined,
};
