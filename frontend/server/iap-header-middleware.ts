import type * as express from 'express';
import { OAuth2Client } from 'google-auth-library';

const TOKEN_HEADER = 'X-Goog-IAP-JWT-Assertion';
const ISSUERS = ['https://cloud.google.com/iap'];

export default class IapHeaderMiddleware {
  audience: string;
  exemptUrls: string[];
  client = new OAuth2Client();
  publicKeys: Record<string, string>;

  constructor(audience: string, exemptUrls: string[]) {
    this.audience = audience;
    this.exemptUrls = exemptUrls;
  }

  async middleware(req: express.Request, res: express.Response, next: express.NextFunction) {
    if (this.exemptUrls.includes(req.originalUrl)) {
      next();
      return;
    }
    const jwt = req.header(TOKEN_HEADER) || '';
    if (jwt === '') {
      const err = new Error(`Required header ${TOKEN_HEADER} not set`);
      next(err);
    }
    try {
      await this.verifyWithRetry(jwt);
    } catch (err) {
      next(err);
    }
    next();
  };

  /**
   * As unofficially recommended by a Google IAP engineer, refresh IAP public keys
   * when key lookup fails: https://stackoverflow.com/a/44897043.
   */
  async verifyWithRetry(jwt: string) {
    if (Object.keys(this.publicKeys).length === 0) {
      this.refreshPublicKeys();
    }
    try {
      return await this.verify(jwt);
    } catch (err) {
      if (err.message === '') {
        this.refreshPublicKeys();
        return await this.verify(jwt);
      } else {
        throw err;
      }
    }
  }

  async verify(jwt: string) {
    return await this.client.verifySignedJwtWithCertsAsync(
      jwt,
      this.publicKeys,
      this.audience,
      ISSUERS
    );
  }

  async refreshPublicKeys() {
    this.publicKeys = (await this.client.getIapPublicKeysAsync()).pubkeys;
  }
};
