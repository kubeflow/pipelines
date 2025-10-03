from kfp import dsl, compiler
import os
import tempfile

with tempfile.TemporaryDirectory() as tmpdir:
    tmpdir_path = os.path.join(tmpdir, "artifact")
    os.makedirs(tmpdir_path, exist_ok=True)
    with open(os.path.join(tmpdir_path, "log.txt"), "w", encoding="utf-8") as f:
        f.write("Hello, world!")

    @dsl.component(embedded_artifact_path=tmpdir_path)
    def read_embedded_artifact_dir(artifact: dsl.EmbeddedInput[dsl.Dataset]):
        import os

        with open(os.path.join(artifact.path, "log.txt"), "r", encoding="utf-8") as f:
            log = f.read()

        assert log == "Hello, world!"


    @dsl.component(embedded_artifact_path=os.path.join(tmpdir_path, "log.txt"))
    def read_embedded_artifact_file(artifact: dsl.EmbeddedInput[dsl.Dataset]):
        with open(artifact.path, "r", encoding="utf-8") as f:
            log = f.read()

        assert log == "Hello, world!"

    @dsl.pipeline(name="nb-simple")
    def pipeline():
        read_embedded_artifact_dir().set_caching_options(False)
        read_embedded_artifact_file().set_caching_options(False)

    if __name__ == "__main__":
        compiler.Compiler().compile(
            pipeline_func=pipeline, package_path=__file__.replace(".py", ".yaml")
        )
