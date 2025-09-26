import json
import os

from kfp import dsl, compiler


NB_DIR = os.path.join(os.path.dirname(__file__), "notebooks")


@dsl.component
def preprocess(text: str, dataset: dsl.Output[dsl.Dataset]):
    import re

    cleaned_text = re.sub(r"\s+", " ", text).strip()
    with open(dataset.path, "w", encoding="utf-8") as f:
        f.write(cleaned_text)


@dsl.notebook_component(
    notebook_path=os.path.join(NB_DIR, "nb_train_with_params.ipynb")
)
def train_model(cleaned_text: dsl.Input[dsl.Dataset], model: dsl.Output[dsl.Model]):
    import shutil

    with open(cleaned_text.path, "r", encoding="utf-8") as f:
        cleaned_text = f.read()

    dsl.run_notebook(cleaned_text=cleaned_text)

    # Notebook writes its model into /tmp/kfp_nb_outputs/model.txt
    shutil.copy("/tmp/kfp_nb_outputs/model.txt", model.path)

    with open(model.path, "r", encoding="utf-8") as f:
        model_text = f.read()

    assert model_text == cleaned_text.upper()


@dsl.notebook_component(notebook_path=os.path.join(NB_DIR, "nb_eval_metrics.ipynb"))
def evaluate_model(model_text: dsl.Input[dsl.Model], metrics: dsl.Output[dsl.Metrics]):
    import json

    with open(model_text.path, "r", encoding="utf-8") as f:
        model_text = f.read()

    dsl.run_notebook(model_text=model_text)
    with open("/tmp/kfp_nb_outputs/metrics.json", "r", encoding="utf-8") as f:
        metrics_dict = json.load(f)

    assert metrics_dict == {"score": float(len(model_text))}

    for metric_name, metric_value in metrics_dict.items():
        metrics.log_metric(metric_name, metric_value)


@dsl.pipeline(name="nb-mixed")
def pipeline(text: str = "Hello   world"):
    p = preprocess(text=text).set_caching_options(False)
    t = train_model(cleaned_text=p.output).set_caching_options(False)
    evaluate_model(model_text=t.output).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace(".py", ".yaml")
    )
