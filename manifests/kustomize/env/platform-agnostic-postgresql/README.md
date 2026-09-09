# PostgreSQL connection configuration

`platform-agnostic-postgresql` and
`platform-agnostic-multi-user-postgresql` require the deploying operator to
explicitly choose the PostgreSQL `sslmode`. The overlays intentionally do not
default this choice: an omitted value makes the API server and cache server
fail before connecting to PostgreSQL, rather than silently using plaintext.

Create an overlay that references the desired PostgreSQL base and merges the
connection parameters into `pipeline-install-config`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../path/to/manifests/kustomize/env/platform-agnostic-postgresql

configMapGenerator:
  - name: pipeline-install-config
    behavior: merge
    literals:
      - postgresExtraParams={"sslmode":"disable"}
```

The `disable` example is appropriate only when the operator deliberately
accepts an unencrypted database connection, such as in local development or a
separately protected network. Do not add it to the production base overlay.

For a TLS-protected PostgreSQL connection, set `sslmode` to an appropriate
value such as `verify-full` and include the corresponding PostgreSQL client
parameters (for example, a trusted CA certificate path). The operator is
responsible for configuring PostgreSQL TLS and mounting those certificate files
in the API server and cache server. Certificate provisioning and pod-to-pod TLS
manifests are outside these overlays' scope.

Use the same pattern with
`platform-agnostic-multi-user-postgresql` for a multi-user installation.
