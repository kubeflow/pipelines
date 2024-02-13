# Copyright 2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from contextlib import contextmanager
import json
import logging
import os
from typing import Any, Callable, Dict, Generator, Iterable, Optional, Tuple
from urllib.parse import parse_qs
from urllib.parse import urlparse
from webbrowser import open_new_tab
import wsgiref.simple_server
import wsgiref.util

import google.auth
import google.auth.app_engine
import google.auth.compute_engine.credentials
import google.auth.iam
from google.auth.transport.requests import Request
import google.oauth2.credentials
import google.oauth2.service_account
import requests

IAM_SCOPE = 'https://www.googleapis.com/auth/iam'
OAUTH_TOKEN_URI = 'https://www.googleapis.com/oauth2/v4/token'
LOCAL_KFP_CREDENTIAL = os.path.expanduser('~/.config/kfp/credentials.json')


def get_gcp_access_token() -> Optional[str]:
    """Gets GCP access token for the current Application Default Credentials.

    Returns:
        GCP access token or None, if it is not set. For more information, see
    https://cloud.google.com/sdk/gcloud/reference/auth/application-default/print-access-token.
    """
    token = None
    try:
        creds, _ = google.auth.default(
            scopes=['https://www.googleapis.com/auth/cloud-platform'])
        if not creds.valid:
            auth_req = Request()
            creds.refresh(auth_req)
        if creds.valid:
            token = creds.token
    except Exception as e:
        logging.warning('Failed to get GCP access token: %s', e)
    return token


def get_auth_token(client_id: str, other_client_id: str,
                   other_client_secret: str) -> Tuple[Optional[str], bool]:
    """Gets auth token from default service account or user account.

    Returns:
        Tuple of (ID token or None, if not found, and a boolean
        indicating whether a refresh token has been saved locally)
    """
    is_refresh_token = True
    if os.path.exists(LOCAL_KFP_CREDENTIAL):
        # fetch IAP auth token using the locally stored credentials.
        with open(LOCAL_KFP_CREDENTIAL, 'r') as f:
            credentials = json.load(f)
        if client_id in credentials:
            saved_refresh_token = credentials[client_id].get('refresh_token')
            saved_other_client_id = credentials[client_id].get(
                'other_client_id')
            saved_other_client_secret = credentials[client_id].get(
                'other_client_secret')
            if None not in {
                    saved_refresh_token, saved_other_client_id,
                    saved_other_client_secret
            }:
                return id_token_from_refresh_token(saved_other_client_id,
                                                   saved_other_client_secret,
                                                   saved_refresh_token,
                                                   client_id), is_refresh_token
    if other_client_id is None or other_client_secret is None:
        # fetch IAP auth token: service accounts
        token = get_auth_token_from_sa(client_id)
        is_refresh_token = False
    else:
        # fetch IAP auth token: user account
        # Obtain the ID token for provided Client ID with user accounts.
        # Flow: get authorization code -> exchange for refresh token -> obtain
        # and return ID token
        refresh_token, is_refresh_token = get_refresh_token_from_client_id(
            other_client_id, other_client_secret)
        credentials = {}
        if os.path.exists(LOCAL_KFP_CREDENTIAL):
            with open(LOCAL_KFP_CREDENTIAL, 'r') as f:
                credentials = json.load(f)
        credentials[client_id] = {
            'other_client_id': other_client_id,
            'other_client_secret': other_client_secret
        }

        if is_refresh_token:
            credentials[client_id]['refresh_token'] = refresh_token
        else:
            credentials[client_id]['access_token'] = refresh_token
        # TODO: handle the case when the refresh_token expires, which only
        # happens if the refresh_token is not used once for six months.
        if not os.path.exists(os.path.dirname(LOCAL_KFP_CREDENTIAL)):
            os.makedirs(os.path.dirname(LOCAL_KFP_CREDENTIAL))
        with open(LOCAL_KFP_CREDENTIAL, 'w') as f:
            json.dump(credentials, f)
        token = id_token_from_refresh_token(other_client_id,
                                            other_client_secret, refresh_token,
                                            client_id)
    return token, is_refresh_token


def get_auth_token_from_sa(client_id: str) -> Optional[str]:
    """Gets auth token from default service account.

    Returns:
        Authorization token or None, if not found.
    """
    service_account_credentials = get_service_account_credentials(client_id)
    if service_account_credentials:
        return get_google_open_id_connect_token(service_account_credentials)
    return None


def get_service_account_credentials(
        client_id: str) -> Optional[google.oauth2.service_account.Credentials]:
    """Figure out what environment we're running in and get some preliminary
    information about the service account.

    Args:
        client_id: OAuth client ID.

    Returns:
        OAuth2 credentials or None.
    """
    bootstrap_credentials, _ = google.auth.default(scopes=[IAM_SCOPE])
    if isinstance(bootstrap_credentials, google.oauth2.credentials.Credentials):
        logging.info('Found OAuth2 credentials and skip SA auth.')
        return None
    if isinstance(bootstrap_credentials, google.auth.app_engine.Credentials):
        # import requests_toolbelt.adapters.appengine here for those who run KFP
        # in an environment where urllib3<2.0.0 (https://github.com/kubeflow/pipelines/blob/9f278f3682662b24b46be2d9ef4a783bcc1f9b0c/sdk/python/requirements.in#L25C14-L25C14)
        # is not available, preventing breaks due to https://github.com/kubeflow/pipelines/issues/9326#issuecomment-1535491761
        # whenever the user runs `import kfp`.
        # by putting the import statement here, only those invoking the KFP SDK client
        # from within App Engine are strictly required to have urllib3<2.0.0.
        import requests_toolbelt.adapters.appengine
        requests_toolbelt.adapters.appengine.monkeypatch()
    # For service account's using the Compute Engine metadata service,
    # service_account_email isn't available until refresh is called.
    bootstrap_credentials.refresh(Request())
    signer_email = bootstrap_credentials.service_account_email
    if isinstance(bootstrap_credentials,
                  google.auth.compute_engine.credentials.Credentials):
        # Since the Compute Engine metadata service doesn't expose the service
        # account key, we use the IAM signBlob API to sign instead.
        # In order for this to work:
        #
        # 1. Your VM needs the https://www.googleapis.com/auth/iam scope.
        #    You can specify this specific scope when creating a VM
        #    through the API or gcloud. When using Cloud Console,
        #    you'll need to specify the "full access to all Cloud APIs"
        #    scope. A VM's scopes can only be specified at creation time.
        #
        # 2. The VM's default service account needs the "Service Account Actor"
        #    role. This can be found under the "Project" category in Cloud
        #    Console, or roles/iam.serviceAccountActor in gcloud.
        signer = google.auth.iam.Signer(Request(), bootstrap_credentials,
                                        signer_email)
    else:
        # A Signer object can sign a JWT using the service account's key.
        signer = bootstrap_credentials.signer
    # Construct OAuth 2.0 service account credentials using the signer
    # and email acquired from the bootstrap credentials.
    return google.oauth2.service_account.Credentials(
        signer,
        signer_email,
        token_uri=OAUTH_TOKEN_URI,
        additional_claims={'target_audience': client_id})


def get_google_open_id_connect_token(
    service_account_credentials: google.oauth2.service_account.Credentials
) -> str:
    """Gets an OpenID Connect token issued by Google for the service account.

    This function:
      1. Generates a JWT signed with the service account's private key
         containing a special "target_audience" claim.
      2. Sends it to the OAUTH_TOKEN_URI endpoint. Because the JWT in #1
         has a target_audience claim, that endpoint will respond with
         an OpenID Connect token for the service account -- in other words,
         a JWT signed by *Google*. The aud claim in this JWT will be
         set to the value from the target_audience claim in #1.
    For more information, see
    https://developers.google.com/identity/protocols/OAuth2ServiceAccount
    The HTTP/REST example on that page describes the JWT structure and
    demonstrates how to call the token endpoint. (The example on that page
    shows how to get an OAuth2 access token; this code is using a
    modified version of it to get an OpenID Connect token).

    Returns:
        OAuth ID token.
    """
    service_account_jwt = (
        service_account_credentials._make_authorization_grant_assertion())
    request = google.auth.transport.requests.Request()
    body = {
        'assertion': service_account_jwt,
        'grant_type': google.oauth2._client._JWT_GRANT_TYPE,
    }
    token_response = google.oauth2._client._token_endpoint_request(
        request, OAUTH_TOKEN_URI, body)
    return token_response['id_token']


def get_refresh_token_from_client_id(client_id: str,
                                     client_secret: str) -> Tuple[str, bool]:
    """Obtains the ID token for provided Client ID with user accounts.
    Flow: get authorization code -> exchange for refresh token -> obtain and
    return ID token.

    Args:
        client_id: OAuth client ID.
        client_secret: OAuth client secret.
            See https://console.cloud.google.com/apis/credentials.

    Returns:
        Tuple of (OAuth short-lived access token or long-lived refresh token,
        and a boolean indicating whether the returned token is refresh_token)
    """
    auth_code, redirect_uri = get_auth_code(client_id)
    token, is_refresh_token = get_refresh_token_from_code(
        auth_code, client_id, client_secret, redirect_uri)
    return token, is_refresh_token


def get_auth_code(client_id: str) -> Tuple[str, str]:
    """Retrieves authorization token using Loopback flow.

    Args:
        client_id: OAuth client ID. To retrieve it, visit
          https://console.cloud.google.com/apis/credentials.

    Returns:
        A tuple of (authorization token, redirect_uri parameter).

    Raises:
        ValueError: If the provided authorization_response is empty.
    """
    host = 'localhost'
    port = 9901
    redirect_uri = f'http://{host}:{port}'
    auth_url = ('https://accounts.google.com/o/oauth2/v2/auth?'
                f'client_id={client_id}&response_type=code&'
                'scope=openid%20email&access_type=offline&'
                f'redirect_uri={redirect_uri}')
    authorization_response = None
    if ('SSH_CONNECTION' in os.environ) or ('SSH_CLIENT'
                                            in os.environ) or is_ipython():
        try:
            print(('SSH connection or IPython shell detected. Please follow the'
                   ' instructions below. Otherwise, press CTRL+C if you are not'
                   ' connected via SSH and not using IPython (e.g. Jupyter'
                   ' Notebook).'))
            authorization_response = get_auth_response_ssh(host, port, auth_url)
        except KeyboardInterrupt:
            authorization_response = None
            logging.warning('User pressed CTRL+C. Trying to open browser...')
    if authorization_response is None:
        try:
            print(('Using a local web-server. Please follow the instructions '
                   'below. Otherwise, press CTRL+C to cancel and manually '
                   'copy-paste the response URL.'))
            authorization_response = get_auth_response_local(
                host, port, auth_url)
        except KeyboardInterrupt:
            logging.warning('User pressed CTRL+C. See instructions below.')
            authorization_response = get_auth_response_ssh(host, port, auth_url)
        except OSError as err:
            logging.warning(
                ('%s.\n Error occurred while creating a local web-server. '
                 'Possibly http://%s:%i is allocated to '
                 'another process. Falling back to manual mode. '
                 'See instructions below.'), err, host, port)
            authorization_response = get_auth_response_ssh(host, port, auth_url)
    if authorization_response is None:
        raise ValueError(
            'Authorization response URL is empty. This may be caused by '
            f'corrupted or expired credentials in {LOCAL_KFP_CREDENTIAL}'
            '. Try renaming or moving them to another directory before '
            'running again.')
    token = fetch_auth_token_from_response(authorization_response)
    return token, redirect_uri


def get_auth_response_ssh(host: str, port: int, auth_url: str) -> str:
    """Fetches OAuth authorization response URL for remote SSH connection.

    Args:
        host: Hostname in redirect_uri.
        port: Port in redirect_uri.
        auth_url: OAuth request URL.

    Returns:
        A URL containing authorization code.
    """
    print(auth_url)
    return input(
        'Carefully follow these steps: (1) open the URL above in your'
        ' browser, (2) authenticate and copy a url of the response page'
        f' that starts with http://{host}:{port}..., and (3) paste it'
        ' below:\n')


def get_auth_response_local(host: str, port: int,
                            auth_url: str) -> Optional[str]:
    """Fetches OAuth authorization response URL using a local web-server.

    Args:
        host: Hostname of the server.
        port: Port of the server.
        auth_url: OAuth request URL.

    Returns:
        A URL containing authorization code.
    """
    with get_local_server_app(host, port) as (local_server, wsgi_app):
        open_new_tab(auth_url)
        print(f'Please visit this URL to authorize Kubeflow SDK: {auth_url}')
        print((f'Make sure that http://{host}:{port} is added to Authorized'
               ' redirect URIs for your OAuth 2.0 Client ID. Check it here:'
               ' https://console.cloud.google.com/apis/credentials'))
        local_server.handle_request()
        return wsgi_app.last_request_uri


def get_refresh_token_from_code(auth_code: str, client_id: str,
                                client_secret: str,
                                redirect_uri: str) -> Tuple[str, bool]:
    """Returns refresh or access token from authorization code.

    Args:
        auth_code: OAuth authorization code.
        client_id: OAuth client ID.
        client_secret: OAuth client secret.
        redirect_uri: Redirect uri used to obtain auth_code.

    Returns:
        Tuple of (OAuth short-lived access token or long-lived refresh token,
        and a boolean indicating whether the returned token is refresh_token)

    Raises:
        ValueError: If HTTP request returns
            a requests.exceptions.HTTPError.
    """
    payload = {
        'code': auth_code,
        'client_id': client_id,
        'client_secret': client_secret,
        'redirect_uri': redirect_uri,
        'grant_type': 'authorization_code'
    }
    res = requests.post(OAUTH_TOKEN_URI, data=payload)
    try:
        res.raise_for_status()
    except requests.exceptions.HTTPError as err:
        raise ValueError(
            ('Some HTTPErrors are caused by expired credentials in '
             f'{LOCAL_KFP_CREDENTIAL} Try renaming or moving '
             'them to another directory before running again.')) from err
    parsed_res = json.loads(res.text)
    token = parsed_res.get('refresh_token')
    is_refresh_token = True
    if token is None:
        token = parsed_res.get('access_token')
        is_refresh_token = False
    return str(token), is_refresh_token


def id_token_from_refresh_token(client_id: str, client_secret: str,
                                refresh_token: str, audience: str) -> str:
    """Returns ID token from refresh token.

    Args:
        client_id: OAuth client ID.
        client_secret: OAuth client secret.
        refresh_token: OAuth refresh token.
        audience: OAuth audience.

    Returns:
        OAuth ID token.

    Raises:
        ValueError: If HTTP request returns
            a requests.exceptions.HTTPError.
    """
    payload = {
        'client_id': client_id,
        'client_secret': client_secret,
        'refresh_token': refresh_token,
        'grant_type': 'refresh_token',
        'audience': audience
    }
    res = requests.post(OAUTH_TOKEN_URI, data=payload)
    try:
        res.raise_for_status()
    except requests.exceptions.HTTPError as err:
        raise ValueError(
            ('Some HTTPErrors are caused by expired credentials in '
             f'{LOCAL_KFP_CREDENTIAL}. Try renaming or moving '
             'them to another directory before running again.')) from err
    return str(json.loads(res.text).get('id_token'))


class RedirectWSGIApp:
    """WSGI app to handle the authorization redirect.

    Stores the request URI and displays the given success message.
    """

    def __init__(self, success_message: str) -> None:
        """
        Args:
            success_message: The message to display in the web browser
                the authorization flow is complete.
        """
        self.last_request_uri = None
        self._success_message = success_message

    def __call__(
        self, environ: Dict[str, Any], start_response: Callable[[str, list],
                                                                Callable[...,
                                                                         None]]
    ) -> Iterable[bytes]:
        """WSGI Callable. Updates environment dictionary with parameters
        required for WSGI and returns it.

        Args:
            environ: The WSGI environment.
            start_response: The WSGI start_response
                callable.

        Returns:
            The response body.
        """
        wsgiref.util.setup_testing_defaults(environ)
        start_response('200 OK',
                       [('Content-type', 'text/plain; charset=utf-8')])
        self.last_request_uri = wsgiref.util.request_uri(environ)
        return [self._success_message.encode('utf-8')]


@contextmanager
def get_local_server_app(
    host: str, port: int
) -> Generator[Tuple[wsgiref.simple_server.WSGIServer, RedirectWSGIApp], None,
               None]:
    """Creates a local web-server for given host and port.

    Args:
        host: Hostname of the server.
        port: Port of the server.

    Returns:
        Tuple of (a local server instance, WSGI app that handles
            the authorization redirect).
    """
    success_message = ('Kubeflow SDK authentication is completed.'
                       ' You may close this window now.')
    wsgi_app = RedirectWSGIApp(success_message)
    wsgiref.simple_server.WSGIServer.allow_reuse_address = False
    local_server = wsgiref.simple_server.make_server(
        host,
        port,
        wsgi_app,
        handler_class=wsgiref.simple_server.WSGIRequestHandler)
    try:
        yield local_server, wsgi_app
    finally:
        local_server.server_close()
        del wsgi_app


def fetch_auth_token_from_response(url: str) -> str:
    """Fetches authorization code for OAuth2.0 Loopback flow.

    Args:
        url: A string containing the response URL.

    Returns:
        An access code.

    Raises:
        KeyError: If no authorization code is found in the provided url.
    """
    parsed_url = urlparse(url)
    parsed_query = parse_qs(parsed_url.query)
    access_code = parsed_query.get('code')
    if access_code is None:
        raise KeyError((
            'Authorization code is missing or empty in the response.'
            ' Please, try again or check '
            'https://www.kubeflow.org/docs/distributions/gke/deploy/oauth-setup'
        ))
    if isinstance(access_code, list):
        access_code = str(access_code.pop(0))
    return access_code


def is_ipython() -> bool:
    """Returns whether we are running in notebook."""
    try:
        import IPython
        ipy = IPython.get_ipython()
        if ipy is None:
            return False
    except ImportError:
        return False
    return True
