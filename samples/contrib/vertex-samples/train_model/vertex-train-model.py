from kfp.v2 import dsl

@dsl.component(base_image='python:3.8',packages_to_install=['google-cloud-aiplatform==1.36.0,scikit-learn==1.3.2'])
def vertex_train_model(
    model_name: str,
    project_id: str,
    location: str,
):
    from google.cloud import aiplatform
    from sklearn.linear_model import LinearRegression

    aiplatform.init(
        project=project_id, 
        location=location, 
        staging_bucket=f"gs://{shared_state['model-bucket']}",
    )
    model = LinearRegression()
    aiplatform.save_model(model, model_name)


@dsl.pipeline(name='train-model')
def pipeline_train_model():
    vertex_train_model("test-model", "900000000001", "us-west1")


if __name__ == "__main__":
    from kfp.v2 import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline_train_model,
        package_path='vertex_train_model.json')