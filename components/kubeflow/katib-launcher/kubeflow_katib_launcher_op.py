# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import kfp.dsl as dsl

def kubeflow_studyjob_launcher_op(name, namespace, optimizationtype, objectivevaluename, optimizationgoal, requestcount, metricsnames,
                                  parameterconfigs, nasConfig, workertemplatepath, mcollectortemplatepath, suggestionspec,
                                  studyjob_timeout_minutes, delete=True, output_file='/output.txt', step_name='StudyJob-Launcher'):
    return dsl.ContainerOp(
        name = step_name,
        image = 'liuhougangxa/ml-pipeline-kubeflow-studyjob:latest',
        arguments = [
            '--name', name,
            '--namespace', namespace,
            "--optimizationtype", optimizationtype,
            "--objectivevaluename", objectivevaluename,
            "--optimizationgoal", optimizationgoal,
            "--requestcount", requestcount,
            "--metricsnames", metricsnames,
            "--parameterconfigs", parameterconfigs,
            "--nasConfig", nasConfig,
            "--workertemplatepath", workertemplatepath,
            "--mcollectortemplatepath", mcollectortemplatepath,
            "--suggestionspec", suggestionspec,
            "--outputfile", output_file,
            "--deleteAfterDone", delete,
            '--studyjobtimeoutminutes', studyjob_timeout_minutes,
        ],
        file_outputs = {'hyperparameter': output_file}
    )
