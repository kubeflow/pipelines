// Copyright 2026 The Kubeflow Authors
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
import { describe, it, expect } from 'vitest';
import express from 'express';
import requests from 'supertest';
import {
  _isBlockedTarget,
  _extractUrlFromReferer,
  _trimProxyPrefix,
  _routePathWithReferer,
  _rewritePath,
} from './proxy-middleware.js';
import proxyMiddleware from './proxy-middleware.js';

describe('_isBlockedTarget', () => {
  describe('cloud metadata endpoints', () => {
    it('blocks AWS/GCP/Azure metadata IP', () => {
      expect(_isBlockedTarget('169.254.169.254')).toBe(true);
    });

    it('blocks GCP metadata hostname', () => {
      expect(_isBlockedTarget('metadata.google.internal')).toBe(true);
    });

    it('blocks GCP metadata.goog hostname', () => {
      expect(_isBlockedTarget('metadata.goog')).toBe(true);
    });

    it('blocks Alibaba Cloud metadata IP', () => {
      expect(_isBlockedTarget('100.100.100.200')).toBe(true);
    });

    it('is case-insensitive for metadata hostnames', () => {
      expect(_isBlockedTarget('Metadata.Google.Internal')).toBe(true);
    });
  });

  describe('loopback addresses', () => {
    it('blocks 127.0.0.1', () => {
      expect(_isBlockedTarget('127.0.0.1')).toBe(true);
    });

    it('blocks any 127.x.x.x address', () => {
      expect(_isBlockedTarget('127.255.0.1')).toBe(true);
    });

    it('blocks localhost', () => {
      expect(_isBlockedTarget('localhost')).toBe(true);
    });

    it('blocks LOCALHOST (case-insensitive)', () => {
      expect(_isBlockedTarget('LOCALHOST')).toBe(true);
    });

    it('blocks *.localhost subdomains (RFC 6761)', () => {
      expect(_isBlockedTarget('anything.localhost')).toBe(true);
      expect(_isBlockedTarget('foo.bar.localhost')).toBe(true);
    });

    it('blocks IPv6 loopback ::1', () => {
      expect(_isBlockedTarget('::1')).toBe(true);
    });

    it('blocks bracketed IPv6 loopback [::1]', () => {
      expect(_isBlockedTarget('[::1]')).toBe(true);
    });
  });

  describe('private network ranges', () => {
    it('blocks 10.0.0.0/8', () => {
      expect(_isBlockedTarget('10.0.0.1')).toBe(true);
      expect(_isBlockedTarget('10.255.255.255')).toBe(true);
    });

    it('blocks 172.16.0.0/12', () => {
      expect(_isBlockedTarget('172.16.0.1')).toBe(true);
      expect(_isBlockedTarget('172.31.255.255')).toBe(true);
    });

    it('does not block 172.15.x.x or 172.32.x.x', () => {
      expect(_isBlockedTarget('172.15.0.1')).toBe(false);
      expect(_isBlockedTarget('172.32.0.1')).toBe(false);
    });

    it('blocks 192.168.0.0/16', () => {
      expect(_isBlockedTarget('192.168.0.1')).toBe(true);
      expect(_isBlockedTarget('192.168.255.255')).toBe(true);
    });

    it('blocks 0.0.0.0/8', () => {
      expect(_isBlockedTarget('0.0.0.0')).toBe(true);
    });
  });

  describe('link-local addresses', () => {
    it('blocks 169.254.x.x', () => {
      expect(_isBlockedTarget('169.254.0.1')).toBe(true);
    });

    it('blocks IPv6 link-local fe80::', () => {
      expect(_isBlockedTarget('fe80::1')).toBe(true);
    });
  });

  describe('IPv4-mapped IPv6 (dotted-decimal form)', () => {
    it('blocks ::ffff:127.0.0.1', () => {
      expect(_isBlockedTarget('::ffff:127.0.0.1')).toBe(true);
    });

    it('blocks ::ffff:10.0.0.1', () => {
      expect(_isBlockedTarget('::ffff:10.0.0.1')).toBe(true);
    });

    it('blocks ::ffff:192.168.1.1', () => {
      expect(_isBlockedTarget('::ffff:192.168.1.1')).toBe(true);
    });

    it('blocks ::ffff:169.254.169.254 (cloud metadata)', () => {
      expect(_isBlockedTarget('::ffff:169.254.169.254')).toBe(true);
    });

    it('blocks ::ffff:0.0.0.1', () => {
      expect(_isBlockedTarget('::ffff:0.0.0.1')).toBe(true);
    });
  });

  describe('IPv4-mapped IPv6 (hex form, as normalised by Node.js URL parser)', () => {
    it('blocks ::ffff:7f00:1 (127.0.0.1 in hex)', () => {
      expect(_isBlockedTarget('::ffff:7f00:1')).toBe(true);
    });

    it('blocks the hostname from new URL("http://[::ffff:127.0.0.1]")', () => {
      const hostname = new URL('http://[::ffff:127.0.0.1]').hostname;
      expect(_isBlockedTarget(hostname)).toBe(true);
    });

    it('blocks the hostname from new URL("http://[::ffff:10.0.0.1]")', () => {
      const hostname = new URL('http://[::ffff:10.0.0.1]').hostname;
      expect(_isBlockedTarget(hostname)).toBe(true);
    });

    it('blocks the hostname from new URL("http://[::ffff:192.168.1.1]")', () => {
      const hostname = new URL('http://[::ffff:192.168.1.1]').hostname;
      expect(_isBlockedTarget(hostname)).toBe(true);
    });

    it('blocks the hostname from new URL("http://[::ffff:169.254.169.254]")', () => {
      const hostname = new URL('http://[::ffff:169.254.169.254]').hostname;
      expect(_isBlockedTarget(hostname)).toBe(true);
    });

    it('blocks ::ffff:a00:1 (10.0.0.1 in hex)', () => {
      expect(_isBlockedTarget('::ffff:a00:1')).toBe(true);
    });

    it('blocks ::ffff:c0a8:101 (192.168.1.1 in hex)', () => {
      expect(_isBlockedTarget('::ffff:c0a8:101')).toBe(true);
    });

    it('does not block ::ffff: with public IP in hex (8.8.8.8 = 808:808)', () => {
      expect(_isBlockedTarget('::ffff:808:808')).toBe(false);
    });
  });

  describe('allowed targets', () => {
    it('allows public IPs', () => {
      expect(_isBlockedTarget('8.8.8.8')).toBe(false);
      expect(_isBlockedTarget('203.0.113.1')).toBe(false);
    });

    it('allows public hostnames', () => {
      expect(_isBlockedTarget('example.com')).toBe(false);
      expect(_isBlockedTarget('tensorboard.my-org.com')).toBe(false);
    });

    it('allows Kubernetes service hostnames', () => {
      expect(_isBlockedTarget('my-service.my-namespace.svc.cluster.local')).toBe(false);
    });
  });
});

describe('_extractUrlFromReferer', () => {
  const prefix = '/apis/v1beta1/_proxy/';

  it('extracts URL from referer containing proxy prefix', () => {
    expect(
      _extractUrlFromReferer(
        prefix,
        'http://localhost:3000/apis/v1beta1/_proxy/http%3A%2F%2Fexample.com',
      ),
    ).toBe('http%3A%2F%2Fexample.com');
  });

  it('returns empty string when referer does not contain prefix', () => {
    expect(_extractUrlFromReferer(prefix, 'http://localhost:3000/other')).toBe('');
  });

  it('returns empty string for undefined referer', () => {
    expect(_extractUrlFromReferer(prefix)).toBe('');
  });
});

describe('_trimProxyPrefix', () => {
  const prefix = '/apis/v1beta1/_proxy/';

  it('trims prefix when path starts with it', () => {
    expect(_trimProxyPrefix(prefix, '/apis/v1beta1/_proxy/http://example.com')).toBe(
      'http://example.com',
    );
  });

  it('returns path unchanged when prefix is absent', () => {
    expect(_trimProxyPrefix(prefix, '/other/path')).toBe('/other/path');
  });
});

describe('_routePathWithReferer', () => {
  const prefix = '/apis/v1beta1/_proxy/';

  it('extracts origin from path', () => {
    expect(_routePathWithReferer(prefix, prefix + 'http%3A%2F%2Fexample.com%2Fpath')).toBe(
      'http://example.com',
    );
  });

  it('prepends http:// when scheme is missing', () => {
    expect(_routePathWithReferer(prefix, prefix + 'example.com%2Fpath')).toBe('http://example.com');
  });
});

describe('proxy middleware integration', () => {
  const apisPrefix = '/apis/v1beta1';
  const proxyPrefix = apisPrefix + '/_proxy/';

  function createApp(): express.Application {
    const app = express();
    proxyMiddleware(app, apisPrefix);
    return app;
  }

  it('returns 403 for requests targeting cloud metadata IP', async () => {
    const app = createApp();
    await requests(app)
      .get(proxyPrefix + encodeURIComponent('http://169.254.169.254/latest/meta-data/'))
      .expect(403, 'Proxy target is not allowed.');
  });

  it('returns 403 for requests targeting localhost', async () => {
    const app = createApp();
    await requests(app)
      .get(proxyPrefix + encodeURIComponent('http://localhost:8080/secret'))
      .expect(403, 'Proxy target is not allowed.');
  });

  it('returns 403 for requests targeting private IP 10.0.0.1', async () => {
    const app = createApp();
    await requests(app)
      .get(proxyPrefix + encodeURIComponent('http://10.0.0.1:8080/'))
      .expect(403, 'Proxy target is not allowed.');
  });

  it('returns 403 for IPv4-mapped IPv6 targeting loopback', async () => {
    const app = createApp();
    await requests(app)
      .get(proxyPrefix + encodeURIComponent('http://[::ffff:127.0.0.1]:8080/'))
      .expect(403, 'Proxy target is not allowed.');
  });

  it('returns 400 for unparseable proxy target URL', async () => {
    const app = createApp();
    await requests(app)
      .get(proxyPrefix + '://:::invalid')
      .expect(400, 'Invalid proxy target URL.');
  });
});

describe('_rewritePath', () => {
  const prefix = '/apis/v1beta1/_proxy/';

  it('trims proxy prefix from path', () => {
    expect(_rewritePath(prefix, prefix + 'http%3A%2F%2Fexample.com/path', '')).toBe(
      'http://example.com/path',
    );
  });

  it('appends query string when present', () => {
    expect(_rewritePath(prefix, '/some/path', 'key=value')).toBe('/some/path?key=value');
  });

  it('returns path without query string when query is empty', () => {
    expect(_rewritePath(prefix, '/some/path', '')).toBe('/some/path');
  });
});
