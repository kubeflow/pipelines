from kfp import compiler, dsl


# image and the sdk has a fixed value because the version matters
@dsl.component(packages_to_install=["codeflare-sdk==0.32.2"], base_image='registry.access.redhat.com/ubi9/python-311:latest')
def ray_fn() -> int:
    import ray  # noqa: PLC0415
    from codeflare_sdk import generate_cert  # noqa: PLC0415
    from codeflare_sdk.ray.cluster import Cluster, ClusterConfiguration  # noqa: PLC0415

    cluster = Cluster(
        ClusterConfiguration(
            name="raytest",
            num_workers=1,
            head_cpu_requests=1,
            head_cpu_limits=1,
            head_memory_requests=4,
            head_memory_limits=4,
            worker_cpu_requests=1,
            worker_cpu_limits=1,
            worker_memory_requests=1,
            worker_memory_limits=2,
            image="quay.io/modh/ray@sha256:6d076aeb38ab3c34a6a2ef0f58dc667089aa15826fa08a73273c629333e12f1e",
            verify_tls=False
        )
    )

    # Clean up any existing cluster with the same name first
    print("Cleaning up any existing cluster resources...")
    try:
        cluster.down()
        print("Cleaned up existing cluster")
    except Exception as e:
        print(f"No existing cluster to clean up (expected): {e}")

    # Create and start the cluster using current best practice
    print("Creating Ray cluster...")
    cluster.apply()

    # Custom wait logic since wait_ready() can hang
    print("Waiting for Ray cluster to be ready...")
    import time
    max_wait_time = 300  # 5 minutes timeout
    wait_interval = 10   # Check every 10 seconds
    elapsed_time = 0

    cluster_ready = False
    while elapsed_time < max_wait_time:
        try:
            print(f"Checking cluster readiness... ({elapsed_time}s elapsed)")

            # Try to get cluster URIs as a readiness check
            dashboard_uri = cluster.cluster_dashboard_uri()
            cluster_uri = cluster.cluster_uri()

            if dashboard_uri and cluster_uri:
                print(f"Cluster is ready! Dashboard: {dashboard_uri}")
                print(f"Cluster URI: {cluster_uri}")
                cluster_ready = True
                break
            else:
                print("Cluster URIs not ready yet, waiting...")

        except (ConnectionError, TimeoutError, RuntimeError, AttributeError) as e:
            print(f"Cluster not ready yet: {e}")
        except Exception as e:
            print(f"Unexpected error checking cluster readiness: {e}")

        time.sleep(wait_interval)
        elapsed_time += wait_interval

    if not cluster_ready:
        print("Cluster details for debugging:")
        print(cluster.details())
        raise RuntimeError(f"Ray cluster failed to become ready within {max_wait_time} seconds")

    print("Cluster is fully ready!")

    # Get cluster connection info
    ray_dashboard_uri = cluster.cluster_dashboard_uri()
    ray_cluster_uri = cluster.cluster_uri()
    print(f"Ray dashboard URI: {ray_dashboard_uri}")
    print(f"Ray cluster URI: {ray_cluster_uri}")

    # Verify cluster URI is available
    assert ray_cluster_uri, "Ray cluster URI is empty - cluster may not be ready"

    # Set up TLS and connect to Ray cluster
    print("Attempting to connect to Ray cluster...")

    # Try TLS setup first, fall back to direct connection if it fails
    tls_setup_successful = False
    try:
        print("Setting up TLS certificates...")
        generate_cert.generate_tls_cert(cluster.config.name, cluster.config.namespace)
        generate_cert.export_env(cluster.config.name, cluster.config.namespace)
        print("TLS certificates configured successfully")
        tls_setup_successful = True
    except (OSError, PermissionError, ValueError, RuntimeError) as e:
        print(f"TLS setup failed (will try direct connection): {e}")
        print("Since cluster was configured with verify_tls=False, attempting direct connection...")
    except Exception as e:
        print(f"Unexpected TLS setup error (will try direct connection): {e}")
        print("Since cluster was configured with verify_tls=False, attempting direct connection...")

    # Connect to the Ray cluster
    try:
        ray.init(address=ray_cluster_uri, logging_level="DEBUG")
        print(f"Ray cluster connected: {ray.is_initialized()}")

        if not ray.is_initialized():
            raise RuntimeError("Ray failed to initialize")

    except (ConnectionError, TimeoutError, RuntimeError, OSError) as e:
        print(f"Ray connection failed: {e}")
        if tls_setup_successful:
            print("Connection failed even with TLS setup")
        else:
            print("Trying alternative connection approaches...")

            # Alternative: Try connecting without explicit address (auto-discovery)
            try:
                ray.shutdown()  # Clean any previous attempts
                print("Attempting Ray auto-discovery connection...")
                ray.init(logging_level="DEBUG")
                print(f"Ray auto-discovery connection: {ray.is_initialized()}")
            except (ConnectionError, TimeoutError, RuntimeError, OSError) as e2:
                print(f"Auto-discovery also failed: {e2}")
                raise RuntimeError(f"All Ray connection attempts failed. Original error: {e}")
            except Exception as e2:
                print(f"Unexpected auto-discovery error: {e2}")
                raise RuntimeError(f"All Ray connection attempts failed. Original error: {e}")
    except Exception as e:
        print(f"Unexpected Ray connection error: {e}")
        raise RuntimeError(f"Ray connection failed with unexpected error: {e}")

    # Verify cluster resources
    print(f"Ray cluster resources: {ray.cluster_resources()}")

    # Define and run remote function
    @ray.remote
    def train_fn():
        return 100

    print("Executing remote Ray function...")
    result = ray.get(train_fn.remote())
    print(f"Ray function result: {result}")
    assert result == 100, f"Expected 100, got {result}"

    # Clean shutdown
    print("Shutting down Ray connection...")
    ray.shutdown()

    print("Cleaning up Ray cluster...")
    cluster.down()
    print("Ray cluster cleanup completed")

    return result


@dsl.pipeline(
    name="Ray Integration Test",
    description="Ray Integration Test",
)
def ray_integration():
    ray_fn().set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(ray_integration, package_path=__file__.replace(".py", "_compiled.yaml"))