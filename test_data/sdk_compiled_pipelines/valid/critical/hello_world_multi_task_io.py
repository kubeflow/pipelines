# Copyright 2024 The Kubeflow Authors
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
"""Enhanced hello world pipeline demonstrating multi-task workflow with inputs and outputs.

This pipeline improves test coverage by:
- Accepting pipeline-level inputs
- Passing data between multiple tasks
- Producing pipeline-level outputs
- Using both container and Python components
"""

from kfp import compiler
from kfp import dsl


@dsl.component(base_image="public.ecr.aws/docker/library/python:3.12")
def generate_greeting(name: str, greeting_type: str = "formal") -> str:
    """Generate a personalized greeting message.
    
    Args:
        name: Name of the recipient
        greeting_type: Type of greeting ("formal" or "casual")
    
    Returns:
        Formatted greeting message
    """
    if greeting_type == "formal":
        return f"Dear {name}, welcome to Kubeflow Pipelines!"
    else:
        return f"Hey {name}! Welcome to KFP!"


@dsl.container_component
def display_greeting(message: dsl.Input[str], output_file: dsl.Output[dsl.Dataset]):
    """Display greeting and save to output file.
    
    Args:
        message: The greeting message to display
        output_file: Output artifact containing the greeting
    """
    return dsl.ContainerSpec(
        image='registry.access.redhat.com/ubi9/python-311:latest',
        command=['sh', '-c'],
        args=[
            f'echo "Processing greeting..." && '
            f'echo "{message}" | tee {output_file.path} && '
            f'echo "Greeting saved to {output_file.path}"'
        ]
    )


@dsl.component(base_image="public.ecr.aws/docker/library/python:3.12")
def count_words(text: str) -> int:
    """Count words in the given text.
    
    Args:
        text: Input text to analyze
    
    Returns:
        Number of words in the text
    """
    word_count = len(text.split())
    print(f"Word count: {word_count}")
    return word_count


@dsl.pipeline(
    name='hello-world-multi-task-io',
    description='Enhanced hello world with multiple tasks, inputs, and outputs'
)
def hello_world_pipeline(
    recipient_name: str = 'Kubeflow User',
    greeting_style: str = 'formal'
) -> str:
    """Multi-task pipeline demonstrating data flow between components.
    
    Args:
        recipient_name: Name of the person to greet
        greeting_style: Style of greeting ("formal" or "casual")
    
    Returns:
        Final greeting message
    """
    # Task 1: Generate greeting message
    greeting_task = generate_greeting(
        name=recipient_name,
        greeting_type=greeting_style
    )
    
    # Task 2: Display and save greeting (depends on Task 1)
    display_task = display_greeting(
        message=greeting_task.output
    )
    
    # Task 3: Count words in greeting (depends on Task 1)
    count_task = count_words(
        text=greeting_task.output
    )
    
    # Return the final greeting as pipeline output
    return greeting_task.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=hello_world_pipeline,
        package_path=__file__.replace('.py', '.yaml')
    )
