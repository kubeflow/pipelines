from kfp import dsl, compiler
import os


NB_DIR = os.path.join(os.path.dirname(__file__), "notebooks")


@dsl.notebook_component(notebook_path=os.path.join(NB_DIR, "nb_train_simple.ipynb"))
def run_train_notebook(text: str):
    # text is not defined in the notebook but text2 is defined
    dsl.run_notebook(text=text)

    with open("/tmp/kfp_nb_outputs/log.txt", "r", encoding="utf-8") as f:
        log = f.read()

    assert log == text + " " + "default2"


@dsl.pipeline(name="nb-simple")
def pipeline(text: str = "hello"):
    run_train_notebook(text=text).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace(".py", ".yaml")
    )
