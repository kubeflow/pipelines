# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import re

def snake_to_camel(name):
    """Convert snake case to camel case."""
    overrides = {
        "role_arn": "roleARN",
        "execution_role_arn" : "executionRoleARN"
    }
    if name in overrides:
        return overrides[name]
    temp = name.split("_")
    return temp[0] + "".join(ele.title() for ele in temp[1:])

def is_ack_requeue_error(error_msg):
    # ^(start) [one or more alphanumeric characters] in [one or more alphanumeric characters] state cannot be modified or deleted. $(end)
    # Alternative was using substring "state cannot be modified or deleted.", but there is a risk of validation errors including it too.
    requeue_regex =  re.compile(r"^(\w+) in (\w+) state cannot be modified or deleted.$")
    matches = requeue_regex.fullmatch(error_msg)
    if matches is not None:
        return True
    return False