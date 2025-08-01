
$9b187b86-7c0a-42ae-a0bc-2a746b6eb7a3$e15dc3ec-b45e-4cc7-bb07-e76b5dbce99a*v1.0.0 Production Data Processing Pipeline"?First stable version of the production data processing pipeline*��ʬ20
.gs://my-bucket/pipelines/pipeline1-v1.0.0.yaml:�

M

components?*=
;
comp-hello-world'*%
#
executorLabelexec-hello-world
�
deploymentSpec�*�
�
	executors�*�
�
exec-hello-world�*�
�
	container�*�
O
argsG2E
--executor_input
{{$}}
--function_to_execute
hello_world
�
command�2�
sh
-c
��
if ! [ -x "$(command -v pip)" ]; then
    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip
fi

PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location 'kfp==2.14.0' '--no-deps' 'typing-extensions>=3.7.4,<5; python_version<"3.9"' && "$0" "$@"

sh
-ec
��program_path=$(mktemp -d)

printf "%s" "$0" > "$program_path/ephemeral_component.py"
_KFP_RUNTIME=true python3 -m kfp.dsl.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"

{y
import kfp
from kfp import dsl
from kfp.dsl import *
from typing import *

def hello_world():
    print("hello world")



image
python:3.9
2
pipelineInfo"* 

namepipeline-hello-world
�
root�*�
�
dag�*�
�
tasks�*�
~
hello-worldo*m

cachingOptions* 
.
componentRef*

namecomp-hello-world
%
taskInfo*

namehello-world

schemaVersion2.1.0


sdkVersion
kfp-2.14.0B(&This is a successful pipeline version.J1https://github.com/org/repo/pipeline1/tree/v1.0.0Rpipelineversion1