import { Request, Response } from 'express';
import portableFetch from 'portable-fetch';
import { AuthConfigs } from '../configs';
import {
  AuthorizeRequestResources,
  AuthorizeRequestVerb,
  AuthServiceApi,
} from '../src/generated/apis/auth';
import { parseError, ErrorDetails } from '../utils';

export type AuthorizeFn = (
  {
    resources,
    verb,
    namespace,
  }: {
    resources: AuthorizeRequestResources;
    verb: AuthorizeRequestVerb;
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
  // TODO: Use portable-fetch instead of node-fetch in other parts too. The generated api here only
  // supports portable-fetch.
  const authService = new AuthServiceApi(
    { basePath: apiServerAddress },
    undefined,
    portableFetch as any,
  );
  const authorize: AuthorizeFn = async ({ resources, verb, namespace }, req) => {
    if (!authConfigs.enabled) {
      return undefined;
    }
    try {
      // Resources and verb are string enums, they are used as string here, that
      // requires a force type conversion. If we generated client should accept
      // enums instead.
      await authService.authorize(namespace, resources as any, verb as any, {
        // Pass authentication header.
        headers: {
          [authConfigs.kubeflowUserIdHeader]: req.headers[authConfigs.kubeflowUserIdHeader],
        },
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
