# Kubeflow Pipelines on Amazon RDS with IAM database authentication

This overlay runs Kubeflow Pipelines against an external Amazon RDS or Aurora
MySQL database, with the API server authenticating using short-lived IAM tokens
instead of a stored password.

The API server signs a fifteen-minute token with its own IAM identity and
presents it as the MySQL password over TLS. Nothing durable is stored and the
credential expires on its own, so no long-lived database password is needed
once the rollout is confirmed.

The API server is the only component that reaches the database.

## Prerequisites

None of these are created by this overlay.

### 1. The database

Enable IAM authentication on the cluster, then create the schemas and the
database users. Connect as the master user:

```sql
CREATE DATABASE mlpipeline;

-- The IAM-authenticated user. It is granted no CREATE DATABASE privilege;
-- the schema above is created by the operator, not by Kubeflow Pipelines.
CREATE USER 'kfp_apiserver'@'%' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';
GRANT ALL PRIVILEGES ON mlpipeline.* TO 'kfp_apiserver'@'%';

FLUSH PRIVILEGES;
```

### 2. IAM policy and roles

Grant `rds-db:connect` for one database user, scoped to the cluster's resource
id — not the instance identifier, and not a wildcard:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": "rds-db:connect",
    "Resource": "arn:aws:rds-db:<region>:<account>:dbuser:<cluster-resource-id>/kfp_apiserver"
  }]
}
```

Create a role with this policy, trusted by the `ml-pipeline` Kubernetes service
account through IRSA or EKS Pod Identity.

### 3. The database CA bundle

The API server refuses to start with IAM authentication enabled and no CA
bundle, because the token is sent using the cleartext password plugin. Create
the ConfigMap the deployment mounts:

```bash
curl -o global-bundle.pem https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem
kubectl create configmap db-ca-bundle -n kubeflow --from-file=ca.pem=global-bundle.pem
```

## Configure and install

Three files need your values:

| File | What to change |
| --- | --- |
| `params.env` | The RDS endpoint, port and region |
| `db-users-secret.yaml` | The IAM database user for each component |
| `patches/*-sa.yaml` | The account id and role name for each service account |

`mysql-secret` stays in place. Its password is unused once this is on -- both
components authenticate with a token, and each logs a warning if a password is
still configured -- but it is what a rollback to password authentication needs,
so keep it until you are sure you will not want one. The deployments reference
it optionally, so removing it will not stop pods from starting.

Then:

```bash
kubectl apply -k manifests/kustomize/env/aws
```

## Verify

```bash
kubectl -n kubeflow logs deploy/ml-pipeline | grep -i "DB connection\|access denied\|token"
```

The startup line should name the provider and the bundle:

```
DB connection: driver=mysql, credentials="aws-iam" provider, TLS=verified against /etc/db-tls/ca.pem
```

Then upload and run a pipeline, and confirm a second run of the same pipeline
hits the cache.

To confirm tokens are minted per connection rather than once at startup, the
connection has to be a new one. A query that succeeds after fifteen minutes
proves nothing on its own: an already-authenticated session keeps working
whether or not credential refresh is healthy, because the server checks
credentials only during the handshake.

`ConMaxLifeTime` defaults to `120s`, so every pooled connection is recycled well
inside the fifteen-minute token lifetime. Leave the deployment idle past that
lifetime, then run a pipeline: every connection it uses was opened with a token
minted after the original one expired.

## Roll back

Rolling back takes two changes, not one. The switch selects how the API server
authenticates; it does not select *who* it authenticates as, and this overlay
points it at an IAM-only database user.

1. Set `dbCredentialProviderEnabled=false` in `params.env`.
2. Set `apiserverUser` in `db-users-secret.yaml` back to the native MySQL user
   whose password is in `mysql-secret`.

Then reapply and restart the deployment. The provider settings can stay in
place.

Changing only the switch leaves the API server presenting `kfp_apiserver` -- a
user created with `AWSAuthenticationPlugin`, and therefore without a password --
to a password login, which fails with access denied.

Keep the native MySQL user, its password, and `mysql-secret` until the rollout
is confirmed. Removing any of them removes this path.
