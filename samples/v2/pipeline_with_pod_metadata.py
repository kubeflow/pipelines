from kfp import dsl
from kfp.compiler import compiler
from kfp.kubernetes import add_pod_annotation
from kfp.kubernetes import add_pod_label
from kfp.kubernetes import use_field_path_as_env


# Component is specific to task_a() and validates that task_a() contains one annotation and two labels.
@dsl.component
def validate_pod_metadata_task_a(annotation_path: str,
                                 annotation_exp_val: str,
                                 label_path_1: str,
                                 label_exp_val_1: str,
                                 label_path_2: str,
                                 label_exp_val_2: str) -> bool:
    import os
    annotation_val = os.getenv(annotation_path)
    if annotation_val != annotation_exp_val:
        raise ValueError(
            f'Pod annotation is {annotation_val} but is supposed to be {annotation_exp_val}.'
        )

    label_val_1 = os.getenv(label_path_1)
    if label_val_1 != label_exp_val_1:
        raise ValueError(
            f'Pod label is {label_val_1} but is supposed to be {label_exp_val_1}.'
        )

    label_val_2 = os.getenv(label_path_2)
    if label_val_2 != label_exp_val_2:
        raise ValueError(
            f'Pod label is {label_val_2} but is supposed to be {label_exp_val_2}.'
        )
    return True

# Component is specific to task_b() and validates that task_b() contains four annotations and three labels.
@dsl.component
def validate_pod_metadata_task_b(annotation_path_1: str,
                                 annotation_exp_val_1: str,
                                 annotation_path_2: str,
                                 annotation_exp_val_2: str,
                                 annotation_path_3: str,
                                 annotation_exp_val_3: str,
                                 annotation_path_4: str,
                                 annotation_exp_val_4: str,
                                 label_path_1: str,
                                 label_exp_val_1: str,
                                 label_path_2: str,
                                 label_exp_val_2: str,
                                 label_path_3: str,
                                 label_exp_val_3: str) -> bool:
    import os
    annotation_val_1 = os.getenv(annotation_path_1)
    if annotation_val_1 != annotation_exp_val_1:
        raise ValueError(
            f'Pod annotation is {annotation_val_1} but is supposed to be {annotation_exp_val_1}.'
        )
    annotation_val_2 = os.getenv(annotation_path_2)
    if annotation_val_2 != annotation_exp_val_2:
        raise ValueError(
            f'Pod annotation is {annotation_val_2} but is supposed to be {annotation_exp_val_2}.'
        )
    annotation_val_3 = os.getenv(annotation_path_3)
    if annotation_val_3 != annotation_exp_val_3:
        raise ValueError(
            f'Pod annotation is {annotation_val_3} but is supposed to be {annotation_exp_val_3}.'
        )
    annotation_val_4 = os.getenv(annotation_path_4)
    if annotation_val_4 != annotation_exp_val_4:
        raise ValueError(
            f'Pod annotation is {annotation_val_4} but is supposed to be {annotation_exp_val_4}.'
        )
    label_val_1 = os.getenv(label_path_1)
    if label_val_1 != label_exp_val_1:
        raise ValueError(
            f'Pod label is {label_val_1} but is supposed to be {label_exp_val_1}.'
        )
    label_val_2 = os.getenv(label_path_2)
    if label_val_2 != label_exp_val_2:
        raise ValueError(
            f'Pod label is {label_val_2} but is supposed to be {label_exp_val_2}.'
        )
    label_val_3 = os.getenv(label_path_3)
    if label_val_3 != label_exp_val_3:
        raise ValueError(
            f'Pod label is {label_val_3} but is supposed to be {label_exp_val_3}.'
        )
    return True

# Component is specific to task_c() and validates that task_c() contains two annotations and zero labels.
@dsl.component
def validate_pod_metadata_task_c(annotation_path_1: str,
                                 annotation_exp_val_1: str,
                                 annotation_path_2: str,
                                 annotation_exp_val_2: str) -> bool:
    import os
    annotation_val_1 = os.getenv(annotation_path_1)
    if annotation_val_1 != annotation_exp_val_1:
        raise ValueError(
            f'Pod annotation is {annotation_val_1} but is supposed to be {annotation_exp_val_1}.'
        )

    annotation_val_2 = os.getenv(annotation_path_2)
    if annotation_val_2 != annotation_exp_val_2:
        raise ValueError(
            f'Pod annotation is {annotation_val_2} but is supposed to be {annotation_exp_val_2}.'
        )
    return True


# Component is specific to task_d() and validates that task_c() contains zero annotations and one label.
@dsl.component
def validate_pod_metadata_task_d(label_path_1: str,
                                 label_exp_val_1: str,
                                 label_path_2: str,
                                 label_exp_val_2: str,
                                 label_path_3: str,
                                 label_exp_val_3: str) -> bool:
    import os
    label_val_1 = os.getenv(label_path_1)
    if label_val_1 != label_exp_val_1:
        raise ValueError(
            f'Pod label is {label_val_1} but is supposed to be {label_exp_val_1}.'
        )
    label_val_2 = os.getenv(label_path_2)
    if label_val_2 != label_exp_val_2:
        raise ValueError(
            f'Pod label is {label_val_2} but is supposed to be {label_exp_val_2}.'
        )
    label_val_3 = os.getenv(label_path_3)
    if label_val_3 != label_exp_val_3:
        raise ValueError(
            f'Pod label is {label_val_3} but is supposed to be {label_exp_val_3}.'
        )
    return True

# Validates that the task contains zero annotations or labels.
@dsl.component
def validate_no_pod_metadata() -> bool:
    import os
    annotation = os.getenv('POD_TASK_ANNOTATION')
    annotation_2 = os.getenv('POD_TASK_ANNOTATION_1')
    if annotation != '' or annotation_2 != '':
        raise ValueError(
            f'Pod annotation is {annotation} but is supposed to be None.'
        )
    label = os.getenv('POD_TASK_LABEL')
    label_2 = os.getenv('POD_TASK_LABEL_1')
    if label != '' or label_2 != '':
        raise ValueError(
            f'Pod label is {label} but is supposed to be None.'
        )
    return True


@dsl.pipeline
def pipeline_with_pod_metadata():
    # tasks a-d are set with different metadata and the separate pods representing each component should have
    # separate, different metadata.

    # task_a is set with one annotation and two labels.
    task_a = validate_pod_metadata_task_a(
        annotation_path = 'POD_TASK_ANNOTATION',
        annotation_exp_val = 'annotation',
        label_path_1 = 'POD_TASK_LABEL_1',
        label_exp_val_1 = 'label-1',
        label_path_2 = 'POD_TASK_LABEL_2',
        label_exp_val_2 = 'label-2').set_caching_options(False)
    add_pod_label(task_a, 'task-label-1', 'label-1')
    add_pod_label(task_a, 'task-label-2', 'label-2')
    add_pod_annotation(task_a, 'task-annotation', 'annotation')
    # expose pod metadata annotation and label in container
    use_field_path_as_env(task_a, 'POD_TASK_ANNOTATION',
                          "metadata.annotations['task-annotation']")
    use_field_path_as_env(task_a, 'POD_TASK_LABEL_1', "metadata.labels['task-label-1']")
    use_field_path_as_env(task_a, 'POD_TASK_LABEL_2', "metadata.labels['task-label-2']")

    # task_b is set with four annotations and three labels.
    task_b = validate_pod_metadata_task_b(
        annotation_path_1 = 'POD_TASK_ANNOTATION_1',
        annotation_exp_val_1 = 'annotation-1',
        annotation_path_2 = 'POD_TASK_ANNOTATION_2',
        annotation_exp_val_2 = 'annotation-2',
        annotation_path_3 = 'POD_TASK_ANNOTATION_3',
        annotation_exp_val_3 = 'annotation-3',
        annotation_path_4 = 'POD_TASK_ANNOTATION_4',
        annotation_exp_val_4 = 'annotation-4',
        label_path_1 = 'POD_TASK_LABEL_1',
        label_exp_val_1 = 'label-1',
        label_path_2 = 'POD_TASK_LABEL_2',
        label_exp_val_2 = 'label-2',
        label_path_3 = 'POD_TASK_LABEL_3',
        label_exp_val_3 = 'label-3'
    ).set_caching_options(False)
    add_pod_annotation(task_b, 'task-annotation-1', 'annotation-1')
    add_pod_annotation(task_b, 'task-annotation-2', 'annotation-2')
    add_pod_annotation(task_b, 'task-annotation-3', 'annotation-3')
    add_pod_annotation(task_b, 'task-annotation-4', 'annotation-4')
    add_pod_label(task_b, 'task-label-1', 'label-1')
    add_pod_label(task_b, 'task-label-2', 'label-2')
    add_pod_label(task_b, 'task-label-3', 'label-3')
    use_field_path_as_env(task_b, 'POD_TASK_ANNOTATION_1',
                          "metadata.annotations['task-annotation-1']")
    use_field_path_as_env(task_b, 'POD_TASK_ANNOTATION_2',
                          "metadata.annotations['task-annotation-2']")
    use_field_path_as_env(task_b, 'POD_TASK_ANNOTATION_3',
                          "metadata.annotations['task-annotation-3']")
    use_field_path_as_env(task_b, 'POD_TASK_ANNOTATION_4',
                          "metadata.annotations['task-annotation-4']")
    use_field_path_as_env(task_b, 'POD_TASK_LABEL_1', "metadata.labels['task-label-1']")
    use_field_path_as_env(task_b, 'POD_TASK_LABEL_2', "metadata.labels['task-label-2']")
    use_field_path_as_env(task_b, 'POD_TASK_LABEL_3', "metadata.labels['task-label-3']")

    # task_c is set with two annotations and zero labels.
    task_c = validate_pod_metadata_task_c(
        annotation_path_1 = 'POD_TASK_ANNOTATION_1',
        annotation_exp_val_1 = 'annotation-1',
        annotation_path_2 = 'POD_TASK_ANNOTATION_2',
        annotation_exp_val_2 = 'annotation-2').set_caching_options(False)
    add_pod_annotation(task_c, 'task-annotation-1', 'annotation-1')
    add_pod_annotation(task_c, 'task-annotation-2', 'annotation-2')
    use_field_path_as_env(task_c, 'POD_TASK_ANNOTATION_1', "metadata.annotations['task-annotation-1']")
    use_field_path_as_env(task_c, 'POD_TASK_ANNOTATION_2', "metadata.annotations['task-annotation-2']")

    # task_d is set with zero annotations and three labels.
    task_d = validate_pod_metadata_task_d(
        label_path_1 = 'POD_TASK_LABEL_1',
        label_exp_val_1 = 'label-1',
        label_path_2 = 'POD_TASK_LABEL_2',
        label_exp_val_2 = 'label-2',
        label_path_3 = 'POD_TASK_LABEL_3',
        label_exp_val_3 = 'label-3',
    ).set_caching_options(False)
    add_pod_label(task_d, 'task-label-1', 'label-1')
    add_pod_label(task_d, 'task-label-2', 'label-2')
    add_pod_label(task_d, 'task-label-3', 'label-3')
    use_field_path_as_env(task_d, 'POD_TASK_LABEL_1', "metadata.labels['task-label-1']")
    use_field_path_as_env(task_d, 'POD_TASK_LABEL_2', "metadata.labels['task-label-2']")
    use_field_path_as_env(task_d, 'POD_TASK_LABEL_3', "metadata.labels['task-label-3']")

# task e is set with no metadata.
    task_e = validate_no_pod_metadata().set_caching_options(False)
    use_field_path_as_env(task_e, 'POD_TASK_ANNOTATION', "metadata.annotations['task-annotation']")
    use_field_path_as_env(task_e, 'POD_TASK_ANNOTATION_1', "metadata.annotations['task-annotation-1']")
    use_field_path_as_env(task_e, 'POD_TASK_LABEL', "metadata.labels['task-label']")
    use_field_path_as_env(task_e, 'POD_TASK_LABEL_1', "metadata.labels['task-label-1']")

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_pod_metadata,
        package_path=__file__.replace('.py', '.yaml'))
