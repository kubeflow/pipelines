// Copyright 2019 Google LLC
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
import fetch from 'node-fetch';

/** IAWSMetadataCredentials describes the credentials provided by aws metadata store. */
export interface IAWSMetadataCredentials {
    Code: string;
    LastUpdated: string;
    Type: string;
    AccessKeyId: string;
    SecretAccessKey: string;
    Token: string;
    Expiration: string;
}

/** url for aws metadata store. */
const metadataUrl = "http://169.254.169.254/latest/meta-data/";


/**
 * Get the AWS IAM instance profile.
 */
async function getIAMInstanceProfile() : Promise<string|undefined> {
    try {
        const resp = await fetch(`${metadataUrl}/iam/security-credentials/`);
        const profiles = (await resp.text()).split('\n');
        if (profiles.length > 0) {
            return profiles[0].trim();  // return first profile
        }
        return;
    } catch (error) {
        console.error(`Unable to fetch credentials from AWS metadata store: ${error}`)
        return;
    }
}

/**
 * Class to handle the session credentials for AWS ec2 instance profile.
 */
class AWSInstanceProfileCredentials {
    _iamProfilePromise = getIAMInstanceProfile();
    _credentials?: IAWSMetadataCredentials;
    _expiration: number = 0;

    async ok() {
        return !!(await this._iamProfilePromise);
    }

    async _fetchCredentials(): Promise<IAWSMetadataCredentials|undefined> {
        try {
            const profile = await this._iamProfilePromise;
            const resp = await fetch(`${metadataUrl}/iam/security-credentials/${profile}`)
            return resp.json();
        } catch (error) {
            console.error(`Unable to fetch credentials from AWS metadata store:${error}`)
            return;
        }
    }

    /**
     * Get the AWS metadata store session credentials.
     */
    async getCredentials(): Promise<IAWSMetadataCredentials> {
        // query for credentials if going to expire or no credentials yet
        if ((Date.now() + 10 >= this._expiration) || !this._credentials) {
            this._credentials = await this._fetchCredentials();
            if (this._credentials.Expiration)
                this._expiration = new Date(this._credentials.Expiration).getTime();
            else 
                this._expiration = -1;  // always expire
        }
        return this._credentials
    }

}

export const awsInstanceProfileCredentials = new AWSInstanceProfileCredentials();