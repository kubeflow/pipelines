from kfp import dsl
from kfp import compiler
from kfp.dsl import Dataset, Model


@dsl.component
def gen_data() -> Dataset:
    dataset = Dataset(uri=dsl.get_uri())
    with open(dataset.path, "w") as f:
        f.write("some data")

    dataset.metadata["length"] = len("some data")
    return dataset


@dsl.component
def train_model(dataset: Dataset) -> Model:
    with open(dataset.path) as f:
        lines = f.read()

    assert lines == "some data"
    assert dataset.metadata["length"] == len("some data")

    model_artifact = Model(uri=dsl.get_uri("model"))
    with open(model_artifact.path, "w") as f:
        f.write("model trained")

    return model_artifact


@dsl.pipeline(name="pythonic-artifacts-test")
def pythonic_artifacts_test_pipeline():
    t1 = gen_data().set_caching_options(False)
    train_model(dataset=t1.output).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pythonic_artifacts_test_pipeline,
        package_path=__file__.replace(".py", ".yaml"),
    )