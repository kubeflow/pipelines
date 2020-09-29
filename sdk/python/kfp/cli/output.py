# Copyright 2018 Google LLC
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

import json
from enum import Enum, unique

from tabulate import tabulate


@unique
class OutputFormat(Enum):
    table = "table"
    json = "json"


def print_output(data, headers, output_format, table_format='simple'):
    if output_format == "table":
        print(tabulate(data, headers=headers, tablefmt=table_format))
    elif output_format == "json":
        if not headers:
            output = data
        else:
            output = []
            for row in data:
                output.append(dict(zip(headers, row)))
        print(json.dumps(output, indent=4))
