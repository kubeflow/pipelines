# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
from kfp import dsl
from kfp import compiler
from kfp import components
import ai_pipeline_params as params
import json


fairness_check_ops = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/master/components/ibm-components/metrics/bias-detector/component.yaml')

@dsl.pipeline(
  name='Katib Run with AIF360',
  description='A pipeline for training using Katib and does bias detection with AIF360'
)
def kfPipeline(
    studyjob_name='ptjob-example-katib-run',
    optimization_type='maximize',
    objective_value_name='accuracy',
    optimization_goal=0.99,
    request_count=1,
    metrics_names='accuracy',
    parameter_configs='[{"name":"--dummy","parametertype":"int","feasible":{"min":"1","max":"2"}}]',
    nas_config='',
    worker_template_path='pytorchWorker.yaml',
    metrics_collector_template_path='',
    suggestion_spec='{"suggestionAlgorithm":"random","requestNumber":1}',
    studyjob_timeout_minutes=120,
    delete_finished_job=True,
    model_class_file='PyTorchModel.py',
    model_class_name='ThreeLayerCNN',
    feature_testset_path='processed_data/X_test.npy',
    label_testset_path='processed_data/y_test.npy',
    protected_label_testset_path='processed_data/p_test.npy',
    favorable_label='0.0',
    unfavorable_label='1.0',
    privileged_groups="[{'race': 0.0}]",
    unprivileged_groups="[{'race': 4.0}]"
):

    def katib_op(studyjob_name, optimization_type, objective_value_name, optimization_goal, metrics_names, request_count=1, namespace="kubeflow",
                 parameter_configs='', nas_config='', worker_template_path='', metrics_collector_template_path='', suggestion_spec='',
                 studyjob_timeout_minutes=10, delete_finished_job=True, output_file='/output.txt', step_name='StudyJob-Launcher'):
        return dsl.ContainerOp(
            name=step_name,
            image='aipipeline/katib-launcher:latest',
            arguments=[
                '--name', studyjob_name,
                '--namespace', namespace,
                "--optimizationtype", optimization_type,
                "--objectivevaluename", objective_value_name,
                "--optimizationgoal", optimization_goal,
                "--requestcount", request_count,
                "--metricsnames", metrics_names,
                "--parameterconfigs", parameter_configs,
                "--nasConfig", nas_config,
                "--workertemplatepath", worker_template_path,
                "--mcollectortemplatepath", metrics_collector_template_path,
                "--suggestionspec", suggestion_spec,
                "--outputfile", output_file,
                "--deleteAfterDone", delete_finished_job,
                '--studyjobtimeoutminutes', studyjob_timeout_minutes,
            ],
            file_outputs={'hyperparameter': output_file}
        )


    katib_run = katib_op(studyjob_name=studyjob_name,
                         optimization_type=optimization_type,
                         objective_value_name=objective_value_name,
                         optimization_goal=optimization_goal,
                         request_count=request_count,
                         metrics_names=metrics_names,
                         parameter_configs=parameter_configs,
                         nas_config=nas_config,
                         worker_template_path=worker_template_path,
                         metrics_collector_template_path=metrics_collector_template_path,
                         suggestion_spec=suggestion_spec,
                         studyjob_timeout_minutes=studyjob_timeout_minutes,
                         delete_finished_job=delete_finished_job)
    fairness_check = fairness_check_ops(model_id='training-example',
                                        model_class_file=model_class_file,
                                        model_class_name=model_class_name,
                                        feature_testset_path=feature_testset_path,
                                        label_testset_path=label_testset_path,
                                        protected_label_testset_path=protected_label_testset_path,
                                        favorable_label=favorable_label,
                                        unfavorable_label=unfavorable_label,
                                        privileged_groups=privileged_groups,
                                        unprivileged_groups=unprivileged_groups).after(katib_run)


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(kfPipeline, __file__ + '.tar.gz')
