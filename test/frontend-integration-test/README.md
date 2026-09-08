# Frontend integration test

This test gets triggered by the end-to-end testing workflows.

## Local run

1. Deploy KFP on a k8s cluster and enable port forwarding. Replace the default namespace `kubeflow` below if needed.

    ```bash
    POD=`kubectl get pods -n kubeflow -l app=ml-pipeline-ui -o jsonpath='{.items[0].metadata.name}'`
    kubectl port-forward -n kubeflow ${POD} 3000:3000 &
    ```

1. Build the container with the tests (supports both amd64 and arm64).
   The Node version is derived from `frontend/.nvmrc`:

    ```bash
    docker build \
      --build-arg NODE_VERSION=$(tr -d 'v' < ../../frontend/.nvmrc) \
      -t kfp-frontend-integration-test:local .
    ```

1. Run the test with enabled networking (**this exposes your local networking stack to the testing container**):

    ```bash
    docker run --net=host kfp-frontend-integration-test:local
    ```

   To run just one spec, set `WDIO_SPECS`:

    ```bash
    docker run --net=host \
      -e WDIO_SPECS=./tensorboard-example.spec.js \
      kfp-frontend-integration-test:local
    ```

   To debug interactively, disable headless mode and enable WDIO logs. The Selenium base image exposes a noVNC desktop on port `7900`, so you can watch the live browser session at `http://localhost:7900`:

    ```bash
    docker run --net=host \
      -e DEBUG=1 \
      -e HEADLESS=false \
      -e WDIO_SPECS=./tensorboard-example.spec.js \
      kfp-frontend-integration-test:local
    ```

   If you're on Docker Desktop (macOS/Windows), use `host.docker.internal` instead of host networking:

    ```bash
    kubectl port-forward --address 0.0.0.0 -n kubeflow ${POD} 3000:3000 &
    docker run -e KFP_BASE_URL=http://host.docker.internal:3000 \
      kfp-frontend-integration-test:local
    ```

   Docker Desktop debug flow:

    ```bash
    kubectl port-forward --address 0.0.0.0 -n kubeflow ${POD} 3000:3000 &
    docker run -p 7900:7900 \
      -e KFP_BASE_URL=http://host.docker.internal:3000 \
      -e DEBUG=1 \
      -e HEADLESS=false \
      -e WDIO_SPECS=./tensorboard-example.spec.js \
      kfp-frontend-integration-test:local
    ```

   If port 4444 is already in use, override Selenium's port:

    ```bash
    docker run -e SELENIUM_PORT=4445 -e SE_OPTS="--port 4445" \
      kfp-frontend-integration-test:local
    ```

## Supported local debug env vars

- `DEBUG=1`: enables WDIO logs.
- `HEADLESS=false`: launches Chromium with the Selenium desktop instead of headless mode.
- `WDIO_SPECS=./tensorboard-example.spec.js`: runs only the listed comma-separated spec paths.
- `FRONTEND_INTEGRATION_SCREENSHOT_DIR=/tmp/frontend-integration-artifacts/screenshots`: overrides debug screenshot output.
- `WDIO_JUNIT_OUTPUT_DIR=/tmp/frontend-integration-artifacts`: overrides the JUnit reporter output path.

1. Once completed, you can kill the background process of port-forwarding:

    ```
    ps aux | grep '[k]ubectl port-forward'
    kill <PID>
    ```

## Dependency compatibility checks

Use Node.js 22.12.0 or newer (the version in `frontend/.nvmrc` is recommended).

```bash
npm install
npm run test:dependencies
```

These checks load the WebdriverIO configuration, confirm that the pinned browser
installer exposes the API WebdriverIO calls, and confirm that our explicit
Selenium connection skips automatic browser and driver installation. They do not
require Selenium, a deployed KFP cluster, or a ZIP extractor. CI runs them once in
the `frontend-integration-dependency-checks` job of `e2e-test-frontend.yml`; they
are not part of `npm test`.

`@puppeteer/browsers` is pinned to 3.2.2, and the override applies the same pin
to `@wdio/utils`, because the 2.x line that `@wdio/utils` declares still depends
on `extract-zip`, which has no patched release for
[GHSA-jmr9-qjv8-65gv](https://github.com/advisories/GHSA-jmr9-qjv8-65gv).
`@wdio/utils` does not declare support for the 3.x major; the checks above stand
in for that support. Two consequences of 3.x are worth knowing: it requires the
Node.js version above, and it made `proxy-agent` optional, so `HTTPS_PROXY` is
ignored on the browser download path. That path never runs here because the
Selenium host is explicit. Remove the pin and the override when `@wdio/utils`
accepts the 3.x major.
