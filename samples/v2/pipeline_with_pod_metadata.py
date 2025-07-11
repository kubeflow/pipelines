from kfp import dsl
from kfp.kubernetes import add_pod_annotation
from kfp.kubernetes import add_pod_label
from kfp.kubernetes import use_field_path_as_env


@dsl.component
def validate_pod_metadata(annotation_expected_val: str,
                          label_expected_val: str) -> bool:
    import os
    annotation_actual_val = os.getenv('POD_TASK_ANNOTATION')
    print(f"Metadata annotation value for key 'task': {annotation_actual_val}")
    label_actual_val = os.getenv('POD_TASK_LABEL')
    print(f"Metadata label value for key 'task': {label_actual_val}")

    if annotation_actual_val is not annotation_expected_val:
        raise ValueError(
            f"Pod annotation is {annotation_actual_val} but is supposed to be {annotation_expected_val}."
        )
    if label_actual_val is not label_expected_val:
        raise ValueError(
            f"Pod label is {label_actual_val} but is supposed to be {label_expected_val}."
        )
    return True


@dsl.pipeline
def pipeline_with_pod_metadata():
    # task_a and task_b are set with different metadata and the separate pods representing each component should have
    # separate, different metadata.
    task_a = validate_pod_metadata(
        annotation_expected_val='a',
        label_expected_val='a').set_caching_options(False)
    add_pod_label(task_a, 'task', 'a')
    add_pod_annotation(task_a, 'task', 'a')
    # expose pod metadata annotation and label in container
    use_field_path_as_env(task_a, 'POD_TASK_ANNOTATION',
                          "metadata.annotations['task']")
    use_field_path_as_env(task_a, 'POD_TASK_LABEL', "metadata.labels['task']")

    task_b = validate_pod_metadata(
        annotation_expected_val='b',
        label_expected_val='b').set_caching_options(False)
    add_pod_label(task_b, 'task', 'b')
    add_pod_annotation(task_b, 'task', 'b')
    use_field_path_as_env(task_b, 'POD_TASK_ANNOTATION',
                          "metadata.annotations['task']")
    use_field_path_as_env(task_b, 'POD_TASK_LABEL', "metadata.labels['task']")
