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


__all__ = [
    'create_component_from_airflow_op',
]


from typing import List

from ._python_op import _func_to_component_spec, _create_task_factory_from_component_spec


_default_airflow_base_image = 'apache/airflow@sha256:7f60cbef6bf92b1f3a5b4e46044911ced39736a8c3858284d3c5a961b3ba8735'


def create_component_from_airflow_op(op_class, base_image=_default_airflow_base_image, result_output_name='Result', variable_output_names=None, xcom_output_names=None, modules_to_capture: List[str] = None):
    component_spec = _create_component_spec_from_airflow_op(op_class, base_image, result_output_name, variable_output_names, xcom_output_names, modules_to_capture)
    task_factory = _create_task_factory_from_component_spec(component_spec)
    return task_factory


def _create_component_spec_from_airflow_op(
    op_class,
    base_image,
    result_output_name='Result',
    variables_dict_output_name='Variables',
    xcoms_dict_output_name='XComs',
    variables_to_output=None,
    xcoms_to_output=None,
    modules_to_capture: List[str] = None,
):
    variables_output_names = variables_to_output or []
    xcoms_output_names = xcoms_to_output or []
    modules_to_capture = modules_to_capture or [op_class.__module__]

    output_names = []
    if result_output_name is not None:
        output_names.append(result_output_name)
    if variables_dict_output_name is not None:
        output_names.append(variables_dict_output_name)
    if xcoms_dict_output_name is not None:
        output_names.append(xcoms_dict_output_name)
    output_names.extend(variables_output_names)
    output_names.extend(xcoms_output_names)

    from collections import namedtuple
    returnType = namedtuple('AirflowOpOutputs', output_names)

    def _run_airflow_op_closure(*op_args, **op_kwargs) -> returnType:
        (result, variables, xcoms) = _run_airflow_op(op_class, *op_args, **op_kwargs)

        output_values = {}

        import json
        if result_output_name is not None:
            output_values[result_output_name] = str(result)
        if variables_dict_output_name is not None:
            output_values[variables_dict_output_name] = json.dumps(variables)
        if xcoms_dict_output_name is not None:
            output_values[xcoms_dict_output_name] = json.dumps(xcoms)
        for name in variables_output_names:
            output_values[name] = variables[name]
        for name in xcoms_output_names:
            output_values[name] = xcoms[name]

        return returnType(**output_values)

    # Hacking the function signature so that correct component interface is generated
    import inspect
    sig = inspect.Signature(
        parameters=inspect.signature(op_class).parameters,
        return_annotation=returnType,
    )
    _run_airflow_op_closure.__signature__ = sig

    return _func_to_component_spec(_run_airflow_op_closure, base_image=base_image, modules_to_capture=modules_to_capture)


def _run_airflow_op(Op, *op_args, **op_kwargs):
    from airflow.utils import db
    db.initdb()

    from datetime import datetime
    from airflow import DAG, settings
    from airflow.models import TaskInstance, Variable, XCom

    dag = DAG(dag_id='anydag', start_date=datetime.now())
    task = Op(*op_args, **op_kwargs, dag=dag, task_id='anytask')
    ti = TaskInstance(task=task, execution_date=datetime.now())
    result = task.execute(ti.get_template_context())

    variables = {var.id: var.val for var in settings.Session().query(Variable).all()}
    xcoms = {msg.key: msg.value for msg in settings.Session().query(XCom).all()}
    return (result, variables, xcoms)
