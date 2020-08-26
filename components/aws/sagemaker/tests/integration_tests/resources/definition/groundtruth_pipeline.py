import kfp
from kfp import components
from kfp import dsl

sagemaker_gt_op = components.load_component_from_file(
    "../../ground_truth/component.yaml"
)


@dsl.pipeline(
    name="SageMaker GroundTruth image classification test pipeline",
    description="SageMaker GroundTruth image classification test pipeline",
)
def ground_truth_test(
    region="",
    ground_truth_train_job_name="",
    ground_truth_label_attribute_name="",
    ground_truth_train_manifest_location="",
    ground_truth_output_location="",
    ground_truth_task_type="",
    ground_truth_worker_type="",
    ground_truth_label_category_config="",
    ground_truth_ui_template="",
    ground_truth_title="",
    ground_truth_description="",
    ground_truth_num_workers_per_object="",
    ground_truth_time_limit="",
    ground_truth_task_availibility="",
    ground_truth_max_concurrent_tasks="",
    role="",
    workteam_arn="",
):

    sagemaker_gt_op(
        region=region,
        role=role,
        job_name=ground_truth_train_job_name,
        label_attribute_name=ground_truth_label_attribute_name,
        manifest_location=ground_truth_train_manifest_location,
        output_location=ground_truth_output_location,
        task_type=ground_truth_task_type,
        worker_type=ground_truth_worker_type,
        workteam_arn=workteam_arn,
        label_category_config=ground_truth_label_category_config,
        ui_template=ground_truth_ui_template,
        title=ground_truth_title,
        description=ground_truth_description,
        num_workers_per_object=ground_truth_num_workers_per_object,
        time_limit=ground_truth_time_limit,
        task_availibility=ground_truth_task_availibility,
        max_concurrent_tasks=ground_truth_max_concurrent_tasks,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(ground_truth_test, __file__ + ".yaml")
