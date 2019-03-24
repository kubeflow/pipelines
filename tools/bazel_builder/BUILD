CONTAINER_VERSION = "5de9890554d173f12a2d152bf4345b1d9e85227f65b97b3e2ca9ca95ebcb70ae"

platform(
    name = "rbe_ubuntu1604",
    constraint_values = [
        "@bazel_tools//platforms:x86_64",
        "@bazel_tools//platforms:linux",
        "@bazel_tools//tools/cpp:clang",
        "@bazel_toolchains//constraints:xenial",
    ],
    remote_execution_properties = """
        properties: {
            name: "container-image"
            value:"docker://gcr.io/ml-pipeline-test/bazel-builder@sha256:%s"
        }
        """ % (CONTAINER_VERSION),
)
