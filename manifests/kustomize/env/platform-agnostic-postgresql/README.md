# Platform-Agnostic PostgreSQL Installation

⚠️ **Current Status: Experimental / Not Yet Released**

PostgreSQL support for Kubeflow Pipelines is currently under development and **not yet available in a stable release**.

## For End Users

This manifest currently references Kubeflow Pipelines version `2.14.3` (inherited from base), which **does not include PostgreSQL support**. Deploying this manifest will result in connection failures.

**We recommend waiting for an official release (e.g., v2.15.0) that includes PostgreSQL support.**

Once PostgreSQL support is included in a stable release:

1. The base images in `manifests/kustomize/base/pipeline/kustomization.yaml` will be updated to the new version
2. This overlay will automatically work without any modifications

## For Advanced Users / Early Adopters

If you want to try PostgreSQL support before the official release, you need to build the images from the `master` branch yourself.

**Note**: Only the API server and cache server need PostgreSQL support. Other components (persistence agent, scheduled workflow controller, etc.) don't directly interact with the database and can use the stable release images.

### Steps:

1. **Build the required images** from the `master` branch:

   ```bash
   cd backend
   make image_api_server
   make image_cache
   ```

2. **Push them to your own registry**:

   ```bash
   docker tag gcr.io/ml-pipeline/api-server:latest your-registry/kfp-api-server:your-tag
   docker tag gcr.io/ml-pipeline/cache-server:latest your-registry/kfp-cache-server:your-tag
   docker push your-registry/kfp-api-server:your-tag
   docker push your-registry/kfp-cache-server:your-tag
   ```

3. **Create a custom overlay** that references this base and overrides the images:

   ```bash
   # Create a new directory for your custom overlay
   mkdir -p my-postgresql-overlay
   cd my-postgresql-overlay
   ```

   Create a `kustomization.yaml`:

   ```yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization

   resources:
     - ../manifests/kustomize/env/platform-agnostic-postgresql

   images:
     - name: ghcr.io/kubeflow/kfp-api-server
       newName: your-registry/kfp-api-server
       newTag: your-tag
     - name: ghcr.io/kubeflow/kfp-cache-server
       newName: your-registry/kfp-cache-server
       newTag: your-tag
   ```

4. **Deploy using your custom overlay**:
   ```bash
   kubectl apply -k my-postgresql-overlay
   ```

This approach follows the same pattern used in CI testing (see `.github/resources/manifests/standalone/postgresql/`)

## Related Files

- **CI testing**: `.github/resources/manifests/standalone/postgresql/` - These manifests ensure CI tests the current branch's code
- **MySQL version**: `manifests/kustomize/env/platform-agnostic/` - The stable MySQL-based installation
