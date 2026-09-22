import { Request } from 'express';
import { AuthConfigs } from '../configs.js';
import {
  AuthorizeResourcesEnum,
  AuthorizeVerbEnum,
  Configuration as AuthConfiguration,
  AuthServiceApi,
} from '../src/generated/apisv2beta1/auth/index.js';
import { parseError, ErrorDetails } from '../utils.js';

export type AuthorizeFn = (
  {
    resources,
    verb,
    namespace,
  }: {
    resources: AuthorizeResourcesEnum;
    verb: AuthorizeVerbEnum;
    namespace: string;
  },
  req: Request,
) => Promise<ErrorDetails | undefined>;

export const getAuthorizeFn = (
  authConfigs: AuthConfigs,
  otherConfigs: {
    apiServerAddress: string;
  },
) => {
  const { apiServerAddress } = otherConfigs;
  const authService = new AuthServiceApi(
    new AuthConfiguration({ basePath: apiServerAddress, fetchApi: fetch as any }),
  );
  const authorize: AuthorizeFn = async ({ resources, verb, namespace }, req) => {
    if (!authConfigs.enabled) {
      return undefined;
    }
    try {
      const rawKubeflowUserId = req.headers[authConfigs.kubeflowUserIdHeader];
      const kubeflowUserId = Array.isArray(rawKubeflowUserId)
        ? rawKubeflowUserId[0]
        : rawKubeflowUserId;
      await authService.authorize(namespace, resources, verb, {
        // Pass authentication header.
        headers: kubeflowUserId
          ? {
              [authConfigs.kubeflowUserIdHeader]: kubeflowUserId,
            }
          : undefined,
      });
      console.debug(`Authorized to ${verb} ${resources} in namespace ${namespace}.`);
      return undefined;
    } catch (err) {
      const details = await parseError(err);
      const message = `User is not authorized to ${verb} ${resources} in namespace ${namespace}: ${details.message}`;
      console.error(message, details.additionalInfo);
      return { message, additionalInfo: details.additionalInfo };
    }
  };
  return authorize;
};
