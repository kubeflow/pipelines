from kfp import dsl
from kfp.dsl import Model, Dataset, importer


@dsl.component
def use_model(model: Model) -> str:
    """Verify a model artifact was imported successfully."""
    import os
    path = model.path
    if os.path.isdir(path):
        files = sorted(os.listdir(path))
        return f"{len(files)} files: {', '.join(files[:5])}"
    with open(path) as f:
        return f.read()[:200]


@dsl.component
def use_dataset(dataset: Dataset) -> int:
    """Verify a dataset artifact was imported successfully."""
    import os
    path = dataset.path
    if os.path.isdir(path):
        return sum(len(files) for _, _, files in os.walk(path))
    with open(path) as f:
        return len(f.readlines())


@dsl.pipeline(
    name="huggingface-importer-test",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
            size='1Gi',
            kubernetes=dsl.KubernetesWorkspaceConfig(
                pvcSpecPatch={
                    'accessModes': ['ReadWriteOnce'],
                    'storageClassName': 'standard',
                },
            ),
        ),
    ),
)
def huggingface_pipeline():
    # Model config import with filtered download (small, fast for CI)
    model_task = importer(
        artifact_uri='huggingface://gpt2?allow_patterns=config.json',
        artifact_class=Model,
        download_to_workspace=True,
    )

    # Dataset README import with filtered download (small, fast for CI)
    dataset_task = importer(
        artifact_uri='huggingface://wikitext?repo_type=dataset&allow_patterns=README.md',
        artifact_class=Dataset,
        download_to_workspace=True,
    )

    # Specific file import (tests file detection logic in URI parser)
    file_task = importer(
        artifact_uri='huggingface://bert-base-uncased/config.json',
        artifact_class=Model,
    )

    # Model with revision (tests revision parsing)
    model_with_revision = importer(
        artifact_uri='huggingface://gpt2/main',
        artifact_class=Model,
    )

    # Complex query parameters (tests parameter handling)
    filtered_model = importer(
        artifact_uri='huggingface://gpt2?allow_patterns=*.bin&ignore_patterns=*.safetensors',
        artifact_class=Model,
    )

    use_model_task = use_model(model=model_task.output)
    use_dataset_task = use_dataset(dataset=dataset_task.output)


if __name__ == '__main__':
    from kfp.compiler import Compiler
    Compiler().compile(huggingface_pipeline, package_path='huggingface_importer.yaml')
