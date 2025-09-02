# SeaweedFS

- [Official documentation](https://github.com/seaweedfs/seaweedfs/wiki)
- [Official repository](https://github.com/seaweedfs/seaweedfs)

SeaweedFS is a simple and highly scalable distributed file system. It has an S3 interface which makes it usable as an object store for kubeflow.

## Prerequisites

- Kubernetes (any recent Version should work)
- You should have `kubectl` available and configured to talk to the desired cluster.
- `kustomize`
- If you installed kubeflow with minio, use the `istio` dir instead of `base` for the kustomize commands.

## Compile manifests

```bash
kubectl kustomize ./base/
```

## Install SeaweedFS

**WARNING**
This replaces the service `minio-service` and will redirect the traffic to seaweedfs.

```bash
# Optional, but recommended to backup existing minio-service
kubectl get -n kubeflow svc minio-service -o=jsonpath='{.metadata.annotations.kubectl\.kubernetes\.io/last-applied-configuration}' > svc-minio-service-backup.json

kubectl kustomize ./base/ | kubectl apply -f -
```

## Verify deployment

Run

```bash
./test.sh
```

With the ready check on the container it already verifies that the S3 starts correctly.
You can then use it with the endpoint at <http://localhost:8333>.
To create access keys open a shell on the pod and use `weed shell` to configure your instance.
Create a user with the command `s3.configure -user <username> -access_key <access-key> -secret-key <secret-key> -actions Read:<my-bucket>/<my-prefix>,Write::<my-bucket>/<my-prefix> -apply`
Documentation for this can also be found [here](https://github.com/seaweedfs/seaweedfs/wiki/Amazon-S3-API).

## Gateway to Remote Object Storage

The Gateway to Remote Object Storage feature allows SeaweedFS to automatically synchronize local storage with remote cloud storage providers (AWS S3, Azure Blob Storage, Google Cloud Storage). This enables:

- **Automatic Bucket Synchronization**: Local new buckets are automatically created in remote storage
- **Bidirectional Sync**: Changes in local storage are uploaded to remote storage
- **Automatic Cleanup**: Local deleted buckets are automatically deleted in remote storage
- **Multi-Cloud Support**: Connect to multiple cloud storage providers simultaneously

### Configure Remote Storage

Remote storage must be configured before using the gateway. Use the `weed shell` to configure remote storage connections:

#### 1. Access SeaweedFS Shell

```bash
kubectl exec -n kubeflow deployment/seaweedfs -it -- weed shell
```

#### 2. Configure Remote Storage

**AWS S3 Configuration:**

```bash
# Configure AWS S3 remote storage
remote.configure -name=aws1 -type=s3 -s3.access_key=YOUR_ACCESS_KEY -s3.secret_key=YOUR_SECRET_KEY -s3.region=us-east-1 -s3.endpoint=s3.amazonaws.com -s3.storage_class="STANDARD"
```

**Azure Blob Storage Configuration:**

```bash
# Configure Azure Blob Storage
remote.configure -name=azure1 -type=azure -azure.account_name=YOUR_ACCOUNT_NAME -azure.account_key=YOUR_ACCOUNT_KEY
```

**Google Cloud Storage Configuration:**

```bash
# Configure Google Cloud Storage
remote.configure -name=gcs1 -type=gcs -gcs.appCredentialsFile=/path/to/service-account-file.json
```

#### 3. View and Manage Configurations

```bash
# List all remote storage configurations
remote.configure

# Delete a configuration
remote.configure -delete -name=aws1
```

### Setup Gateway to Remote Storage

#### Step 1: Mount Existing Remote Buckets (Optional)

If you have existing buckets in remote storage, mount them as local buckets:

```bash
# In weed shell
remote.mount.buckets -remote=aws1 -apply
```

#### Step 2: Start the Remote Gateway

The gateway process continuously monitors local changes and syncs them to remote storage.

**Basic Gateway Setup:**

```bash
# Start the gateway (run this in the SeaweedFS deployment)
kubectl exec -n kubeflow deployment/seaweedfs -- weed filer.remote.gateway -createBucketAt=aws1
```

**Gateway with Random Suffix (for unique bucket names):**

```bash
# Some cloud providers require globally unique bucket names
kubectl exec -n kubeflow deployment/seaweedfs -- weed filer.remote.gateway -createBucketAt=aws1 -createBucketWithRandomSuffix
```

#### Step 3(Optional): Cache Management

Optimize performance by managing cache:

```bash
# In weed shell

# Cache all PDF files in all mounted buckets
remote.cache -include=*.pdf

# Cache all PDF files in a specific bucket
remote.cache -dir=/buckets/some-bucket -include=*.pdf

# Uncache files older than 1 hour and larger than 10KB
remote.uncache -minAge=3600 -minSize=10240
```

### Troubleshooting

**Common Issues:**

- **Configuration not found**: Ensure remote storage is configured before starting gateway
- **Permission denied**: Check cloud storage credentials and permissions
- **Connection timeout**: Verify network connectivity to cloud storage
- **Bucket conflicts**: Use random suffix for globally unique bucket names

**Debug Commands:**

```bash
# Check remote configurations
kubectl exec -n kubeflow deployment/seaweedfs -- weed shell -c "remote.configure"

# Check mounted buckets
kubectl exec -n kubeflow deployment/seaweedfs -- weed shell -c "remote.mount.buckets -remote=aws1"

# Check gateway logs
kubectl logs -n kubeflow deployment/seaweedfs -f
```

## Uninstall SeaweedFS

```bash
kubectl kustomize ./base/ | kubectl delete -f -
# Restore minio-service from backup
kubectl apply -f svc-minio-service-backup.json
```
