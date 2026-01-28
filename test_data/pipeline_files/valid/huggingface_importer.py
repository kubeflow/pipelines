from kfp import dsl
from kfp.dsl import Model, Dataset, importer

@dsl.component
def use_model(model: Model) -> str:
    with open(model.path) as f:
        return f.read()

@dsl.component
def use_dataset(dataset: Dataset) -> int:
    with open(dataset.path) as f:
        return len(f.readlines())

@dsl.pipeline(name="huggingface-importer-test")
def huggingface_pipeline():
    # Basic model import
    model_task = importer(
        artifact_uri='huggingface://gpt2',
        artifact_class=Model
    )
    
    # Dataset import with repo_type parameter
    dataset_task = importer(
        artifact_uri='huggingface://wikitext?repo_type=dataset',
        artifact_class=Dataset
    )
    
    # Specific file import (tests file detection logic)
    file_task = importer(
        artifact_uri='huggingface://bert-base-uncased/config.json',
        artifact_class=Model
    )
    
    # Model with revision (tests revision parsing)
    model_with_revision = importer(
        artifact_uri='huggingface://gpt2/main',
        artifact_class=Model
    )
    
    # Complex query parameters (tests parameter handling)
    filtered_model = importer(
        artifact_uri='huggingface://gpt2?allow_patterns=*.bin&ignore_patterns=*.safetensors',
        artifact_class=Model
    )
    
    # Organization/repo format (tests path parsing)
    org_model = importer(
        artifact_uri='huggingface://meta-llama/Llama-2-7b-chat-hf',
        artifact_class=Model
    )
    
    use_model_task = use_model(model=model_task.output)
    use_dataset_task = use_dataset(dataset=dataset_task.output)


if __name__ == '__main__':
    from kfp.compiler import Compiler
    Compiler().compile(huggingface_pipeline, package_path='huggingface_importer_pipeline.yaml')
