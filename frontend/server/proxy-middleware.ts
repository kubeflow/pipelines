import * as proxy from 'http-proxy-middleware';
import { URL, URLSearchParams } from 'url';

export function _extractUrlFromReferer(proxyPrefix: string, referer: string) {
  const index = referer.indexOf(proxyPrefix);
  return index > -1 ?
    referer.substr(index + proxyPrefix.length) :
    '';
}

export function _trimProxyPrefix(proxyPrefix: string, path: string) {
  return path.indexOf(proxyPrefix) === 0 ?
    path = path.substr(proxyPrefix.length) :
    path;
}

export function _routePathWithReferer(proxyPrefix: string, path: string, referer?: string) {
  // If a referer header is included, extract the referer URL, otherwise
  // just trim out the /_proxy/ prefix. Use the origin of the resulting URL.
  const decodedPath = decodeURIComponent(referer ?
    _extractUrlFromReferer(proxyPrefix, referer) :
    _trimProxyPrefix(proxyPrefix, path));
  return new URL(decodedPath).origin;
}

export function _rewritePath(proxyPrefix: string, path: string, query: string) {
  // Trim the proxy prefix if exists. It won't exist for any requests made
  // to absolute paths by the proxied resource.
  const querystring = new URLSearchParams(query).toString();
  const decodedPath = decodeURIComponent(path);
  return _trimProxyPrefix(proxyPrefix, decodedPath) +
    (querystring && '?' + querystring);
}

export default (app, apisPrefix) => {

  const proxyPrefix = apisPrefix + '/_proxy/';

  app.use((req, res, next) => {
    // For any request that has a proxy referer header but no proxy prefix,
    // prepend the proxy prefix to it and redirect.
    if (req.headers.referer) {
      const refererUrl = _extractUrlFromReferer(proxyPrefix, req.headers.referer);
      if (refererUrl && req.url.indexOf(proxyPrefix) !== 0) {
        const proxiedUrl = decodeURIComponent(
          _extractUrlFromReferer(proxyPrefix, req.headers.referer));
        const proxiedOrigin = new URL(proxiedUrl).origin;
        req.url = proxyPrefix + encodeURIComponent(proxiedOrigin + req.url);
      }
    }
    next();
  });

  app.all(proxyPrefix + '*', proxy({
    changeOrigin: true,
    logLevel: 'debug',
    target: 'http://127.0.0.1',

    router: (req) => {
      return _routePathWithReferer(proxyPrefix, req.path, req.headers.referer)
    },

    pathRewrite: (_, req) => {
      return _rewritePath(proxyPrefix, req.path, req.query);
    },
  }));

}
