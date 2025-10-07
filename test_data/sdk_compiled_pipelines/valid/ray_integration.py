from kfp import compiler, dsl

common_base_image = (
    "registry.redhat.io/ubi8/python-39@sha256:3523b184212e1f2243e76d8094ab52b01ea3015471471290d011625e1763af61"
)


# image and the sdk has a fixed value because the version matters
@dsl.component(packages_to_install=["codeflare-sdk==0.21.1"], base_image=common_base_image)
def ray_fn() -> int:
    import ray  # noqa: PLC0415
    from codeflare_sdk import generate_cert  # noqa: PLC0415
    from codeflare_sdk.cluster.cluster import Cluster, ClusterConfiguration  # noqa: PLC0415

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
            image="quay.io/modh/ray:2.35.0-py39-cu121",
            verify_tls=False
        )
    )

    # always clean the resources
    cluster.down()
    print(cluster.status())
    cluster.up()
    cluster.wait_ready()
    print(cluster.status())
    print(cluster.details())

    ray_dashboard_uri = cluster.cluster_dashboard_uri()
    ray_cluster_uri = cluster.cluster_uri()
    print(ray_dashboard_uri)
    print(ray_cluster_uri)

    # before proceeding make sure the cluster exists and the uri is not empty
    assert ray_cluster_uri, "Ray cluster needs to be started and set before proceeding"

    # reset the ray context in case there's already one.
    ray.shutdown()
    # establish connection to ray cluster
    generate_cert.generate_tls_cert(cluster.config.name, cluster.config.namespace)
    generate_cert.export_env(cluster.config.name, cluster.config.namespace)
    ray.init(address=cluster.cluster_uri(), logging_level="DEBUG")
    print("Ray cluster is up and running: ", ray.is_initialized())

    @ray.remote
    def train_fn():
        return 100

    result = ray.get(train_fn.remote())
    assert 100 == result
    ray.shutdown()
    cluster.down()
    return result


@dsl.pipeline(
    name="Ray Integration Test",
    description="Ray Integration Test",
)
def ray_integration():
    ray_fn().set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(ray_integration, package_path=__file__.replace(".py", "_compiled.yaml"))
