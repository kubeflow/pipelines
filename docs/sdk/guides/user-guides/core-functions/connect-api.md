# Connect the SDK to the API

## Overview

The [Kubeflow Pipelines SDK](https://kubeflow-pipelines.readthedocs.io/en/stable/) provides a Python interface to interact with the Kubeflow Pipelines API.
This guide will show you how to connect the SDK to the Pipelines API in various scenarios.


## Kubeflow Platform

When running Kubeflow Pipelines as part of a multi-user [Kubeflow Platform](https://www.kubeflow.org/docs/started/introduction/#what-is-the-kubeflow-ai-reference-platform), how you authenticate the Pipelines SDK will depend on whether you are running your code __inside__ or __outside__ the cluster.

### **Kubeflow Platform - Inside the Cluster**

<details>
<summary>Click to expand</summary>
<hr>

A [ServiceAccount token volume](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection) can be mounted to a Pod running in the same cluster as Kubeflow Pipelines.
The Kubeflow Pipelines SDK can use this token to authenticate itself with the Kubeflow Pipelines API.

The following Python code will create a `kfp.Client()` using a ServiceAccount token for authentication:

```python
import kfp

# by default, when run from inside a Kubernetes cluster:
#  - the token is read from the `KF_PIPELINES_SA_TOKEN_PATH` path
#  - the host is set to `http://ml-pipeline-ui.kubeflow.svc.cluster.local`
kfp_client = kfp.Client()

# test the client by listing experiments
experiments = kfp_client.list_experiments(namespace="my-profile")
print(experiments)
```

#### ServiceAccount Token Volume

To use the preceding code, you will need to run it from a Pod that has a ServiceAccount token volume mounted.
You may manually add a `volume` and `volumeMount` to your PodSpec or use Kubeflow's [`PodDefaults`](https://github.com/kubeflow/dashboard/tree/main/components/poddefaults-webhooks) to inject the required volume.

__Option 1 - manually add a volume to your PodSpec:__

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: access-kfp-example
spec:
  containers:
  - image: hello-world:latest
    name: hello-world
    env:
      - ## this environment variable is automatically read by `kfp.Client()`
        ## this is the default value, but we show it here for clarity
        name: KF_PIPELINES_SA_TOKEN_PATH
        value: /var/run/secrets/kubeflow/pipelines/token
    volumeMounts:
      - mountPath: /var/run/secrets/kubeflow/pipelines
        name: volume-kf-pipeline-token
        readOnly: true
  volumes:
    - name: volume-kf-pipeline-token
      projected:
        sources:
          - serviceAccountToken:
              path: token
              expirationSeconds: 7200
              ## defined by the `TOKEN_REVIEW_AUDIENCE` environment variable on the `ml-pipeline` deployment
              audience: pipelines.kubeflow.org
```

__Option 2 - use a `PodDefault` to inject the volume:__

```yaml
apiVersion: kubeflow.org/v1alpha1
kind: PodDefault
metadata:
  name: access-ml-pipeline
  namespace: "<YOUR_USER_PROFILE_NAMESPACE>"
spec:
  desc: Allow access to Kubeflow Pipelines
  selector:
    matchLabels:
      access-ml-pipeline: "true"
  env:
    - ## this environment variable is automatically read by `kfp.Client()`
      ## this is the default value, but we show it here for clarity
      name: KF_PIPELINES_SA_TOKEN_PATH
      value: /var/run/secrets/kubeflow/pipelines/token
  volumes:
    - name: volume-kf-pipeline-token
      projected:
        sources:
          - serviceAccountToken:
              path: token
              expirationSeconds: 7200
              ## defined by the `TOKEN_REVIEW_AUDIENCE` environment variable on the `ml-pipeline` deployment
              audience: pipelines.kubeflow.org
  volumeMounts:
    - mountPath: /var/run/secrets/kubeflow/pipelines
      name: volume-kf-pipeline-token
      readOnly: true
```

:::{tip}
* `PodDefaults` are namespaced resources, so you need to create one inside __each__ of your Kubeflow `Profile` namespaces.
* The Notebook Spawner UI will be aware of any `PodDefaults` in the user's namespace (they are selectable under the "configurations" section).
:::

#### RBAC Authorization

The Kubeflow Pipelines API respects Kubernetes RBAC, and will check RoleBindings assigned to the ServiceAccount before allowing it to take Pipelines API actions.

For example, this RoleBinding allows Pods with the `default-editor` ServiceAccount in `namespace-2` to manage Kubeflow Pipelines in `namespace-1`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: allow-namespace-2-kubeflow-edit
  ## this RoleBinding is in `namespace-1`, because it grants access to `namespace-1`
  namespace: namespace-1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kubeflow-edit
subjects:
  - kind: ServiceAccount
    name: default-editor
    ## the ServiceAccount lives in `namespace-2`
    namespace: namespace-2
```

:::{tip}
* Review the ClusterRole called [`aggregate-to-kubeflow-pipelines-edit`](https://github.com/kubeflow/pipelines/blob/efb96135033fc6e6e55078d33814c45a98566e68/manifests/kustomize/base/installs/multi-user/view-edit-cluster-roles.yaml#L36-L99)
for a list of some important `pipelines.kubeflow.org` RBAC verbs.
* Kubeflow Notebooks pods run as the `default-editor` ServiceAccount by default, so the RoleBindings for `default-editor` apply to them
and give them access to submit pipelines in their own namespace.
* For more information about profiles, see the [Manage Profile Contributors](https://www.kubeflow.org/docs/components/central-dash/profiles/#manage-profile-contributors) guide.
:::

</details>

### **Kubeflow Platform - Outside the Cluster**

<details>
<summary>Click to expand</summary>
<hr>

:::{admonition} Kubeflow Notebooks
:class: warning

As Kubeflow Notebooks run on Pods _inside the cluster_, they can NOT use the following method to authenticate the Pipelines SDK, see the [inside the cluster](#kubeflow-platform---inside-the-cluster) method.
:::

The precise method to authenticate from _outside the cluster_ will depend on how you [deployed Kubeflow Platform](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-ai-reference-platform).
Because most distributions use [Dex](https://dexidp.io/) as their identity provider, this example will show you how to authenticate with Dex using a Python script.

You will need to make the Kubeflow Pipelines API accessible on the remote machine.
If your Kubeflow Istio gateway is already exposed, skip this step and use that URL directly.

The following command will expose the `istio-ingressgateway` service on `localhost:8080`:

```bash
# TIP: svc/istio-ingressgateway may be called something else,
#      or use different ports in your distribution
kubectl port-forward --namespace istio-system svc/istio-ingressgateway 8080:80
```

The following Python code defines a `KFPClientManager()` class that creates an authenticated `kfp.Client()` by interacting with Dex:

```python
import re
from urllib.parse import urlsplit, urlencode

import kfp
import requests
import urllib3


class KFPClientManager:
    """
    A class that creates `kfp.Client` instances with Dex authentication.
    """

    def __init__(
        self,
        api_url: str,
        dex_username: str,
        dex_password: str,
        dex_auth_type: str = "local",
        skip_tls_verify: bool = False,
    ):
        """
        Initialize the KfpClient

        :param api_url: the Kubeflow Pipelines API URL
        :param skip_tls_verify: if True, skip TLS verification
        :param dex_username: the Dex username
        :param dex_password: the Dex password
        :param dex_auth_type: the auth type to use if Dex has multiple enabled, one of: ['ldap', 'local']
        """
        self._api_url = api_url
        self._skip_tls_verify = skip_tls_verify
        self._dex_username = dex_username
        self._dex_password = dex_password
        self._dex_auth_type = dex_auth_type
        self._client = None

        # disable SSL verification, if requested
        if self._skip_tls_verify:
            urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

        # ensure `dex_default_auth_type` is valid
        if self._dex_auth_type not in ["ldap", "local"]:
            raise ValueError(
                f"Invalid `dex_auth_type` '{self._dex_auth_type}', must be one of: ['ldap', 'local']"
            )

    def _get_session_cookies(self) -> str:
        """
        Get the session cookies by authenticating against Dex
        :return: a string of session cookies in the form "key1=value1; key2=value2"
        """

        # use a persistent session (for cookies)
        s = requests.Session()

        # GET the api_url, which should redirect to Dex
        resp = s.get(
            self._api_url, allow_redirects=True, verify=not self._skip_tls_verify
        )
        if resp.status_code == 200:
            pass
        elif resp.status_code == 403:
            # if we get 403, we might be at the oauth2-proxy sign-in page
            # the default path to start the sign-in flow is `/oauth2/start?rd=<url>`
            url_obj = urlsplit(resp.url)
            url_obj = url_obj._replace(
                path="/oauth2/start", query=urlencode({"rd": url_obj.path})
            )
            resp = s.get(
                url_obj.geturl(), allow_redirects=True, verify=not self._skip_tls_verify
            )
        else:
            raise RuntimeError(
                f"HTTP status code '{resp.status_code}' for GET against: {self._api_url}"
            )

        # if we were NOT redirected, then the endpoint is unsecured
        if len(resp.history) == 0:
            # no cookies are needed
            return ""

        # if we are at `../auth` path, we need to select an auth type
        url_obj = urlsplit(resp.url)
        if re.search(r"/auth$", url_obj.path):
            url_obj = url_obj._replace(
                path=re.sub(r"/auth$", f"/auth/{self._dex_auth_type}", url_obj.path)
            )

        # if we are at `../auth/xxxx/login` path, then we are at the login page
        if re.search(r"/auth/.*/login$", url_obj.path):
            dex_login_url = url_obj.geturl()
        else:
            # otherwise, we need to follow a redirect to the login page
            resp = s.get(
                url_obj.geturl(), allow_redirects=True, verify=not self._skip_tls_verify
            )
            if resp.status_code != 200:
                raise RuntimeError(
                    f"HTTP status code '{resp.status_code}' for GET against: {url_obj.geturl()}"
                )
            dex_login_url = resp.url

        # attempt Dex login
        resp = s.post(
            dex_login_url,
            data={"login": self._dex_username, "password": self._dex_password},
            allow_redirects=True,
            verify=not self._skip_tls_verify,
        )
        if resp.status_code != 200:
            raise RuntimeError(
                f"HTTP status code '{resp.status_code}' for POST against: {dex_login_url}"
            )

        # if we were NOT redirected, then the login credentials were probably invalid
        if len(resp.history) == 0:
            raise RuntimeError(
                f"Login credentials are probably invalid - "
                f"No redirect after POST to: {dex_login_url}"
            )

        # if we are at `../approval` path, we need to approve the login
        url_obj = urlsplit(resp.url)
        if re.search(r"/approval$", url_obj.path):
            dex_approval_url = url_obj.geturl()

            # approve the login
            resp = s.post(
                dex_approval_url,
                data={"approval": "approve"},
                allow_redirects=True,
                verify=not self._skip_tls_verify,
            )
            if resp.status_code != 200:
                raise RuntimeError(
                    f"HTTP status code '{resp.status_code}' for POST against: {url_obj.geturl()}"
                )

        return "; ".join([f"{c.name}={c.value}" for c in s.cookies])

    def _create_kfp_client(self) -> kfp.Client:
        try:
            session_cookies = self._get_session_cookies()
        except Exception as ex:
            raise RuntimeError(f"Failed to get Dex session cookies") from ex

        # monkey patch the kfp.Client to support disabling SSL verification
        # kfp only added support in v2: https://github.com/kubeflow/pipelines/pull/7174
        original_load_config = kfp.Client._load_config

        def patched_load_config(client_self, *args, **kwargs):
            config = original_load_config(client_self, *args, **kwargs)
            config.verify_ssl = not self._skip_tls_verify
            return config

        patched_kfp_client = kfp.Client
        patched_kfp_client._load_config = patched_load_config

        return patched_kfp_client(
            host=self._api_url,
            cookies=session_cookies,
        )

    def create_kfp_client(self) -> kfp.Client:
        """Get a newly authenticated Kubeflow Pipelines client."""
        return self._create_kfp_client()
```

The following Python code shows how to use the `KFPClientManager()` class to create a `kfp.Client()`:

```python
# initialize a KFPClientManager
kfp_client_manager = KFPClientManager(
    api_url="http://localhost:8080/pipeline",
    skip_tls_verify=True,

    dex_username="user@example.com",
    dex_password="12341234",

    # can be 'ldap' or 'local' depending on your Dex configuration
    dex_auth_type="local",
)

# get a newly authenticated KFP client
# TIP: long-lived sessions might need to get a new client when their session expires
kfp_client = kfp_client_manager.create_kfp_client()

# test the client by listing experiments
experiments = kfp_client.list_experiments(namespace="my-profile")
print(experiments)
```

</details>

## Standalone Kubeflow Pipelines

When running Kubeflow Pipelines in [standalone mode](../../operator-guides/installation/index.md), there will be no concept of multi-user authentication or RBAC.
The specific steps will depend on whether you are running your code __inside__ or __outside__ the cluster.

### **Standalone KFP - Inside the Cluster**

<details>
<summary>Click to expand</summary>
<hr>

When running inside the Kubernetes cluster, you may connect Pipelines SDK directly to the `ml-pipeline-ui` service via [cluster-internal service DNS resolution](https://kubernetes.io/docs/concepts/services-networking/service/#discovering-services).

:::{warning}
In [standalone deployments](../../operator-guides/installation/index.md) of Kubeflow Pipelines, there is no authentication enforced on the `ml-pipeline-ui` service.
:::

When running in the __same namespace__ as Kubeflow:

```python
import kfp

client = kfp.Client(host="http://ml-pipeline-ui:80")

print(client.list_experiments())
```

When running in a __different namespace__ to Kubeflow:

```python
import kfp

# the namespace in which you deployed Kubeflow Pipelines
namespace = "kubeflow"

client = kfp.Client(host=f"http://ml-pipeline-ui.{namespace}")

print(client.list_experiments())
```

</details>

### **Standalone KFP - Outside the Cluster**

<details>
<summary>Click to expand</summary>
<hr>

When running outside the Kubernetes cluster, you may connect Pipelines SDK to the `ml-pipeline-ui` service by using [kubectl port-forwarding](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/).

:::{warning}
In [standalone deployments](../../operator-guides/installation/index.md) of Kubeflow Pipelines, there is no authentication enforced on the `ml-pipeline-ui` service.
:::

__Step 1:__ run the following command on your external system to initiate port-forwarding:

```bash
# change `--namespace` if you deployed Kubeflow Pipelines into a different namespace
kubectl port-forward --namespace kubeflow svc/ml-pipeline-ui 3000:80
```

__Step 2:__ the following code will create a `kfp.Client()` against your port-forwarded `ml-pipeline-ui` service:

```python
import kfp

client = kfp.Client(host="http://localhost:3000")

print(client.list_experiments())
```

</details>

<br>
