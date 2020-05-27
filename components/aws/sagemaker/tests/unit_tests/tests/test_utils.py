# Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You
# may not use this file except in compliance with the License. A copy of
# the License is located at
#
# http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is
# distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF
# ANY KIND, either express or implied. See the License for the specific
# language governing permissions and limitations under the License.

def check_empty_string_values(obj):
    obj_has_empty_string = 0
    if type(obj) is dict:
        for k,v in obj.items():
            if type(v) is str and v == '':
                print(k + '- has empty string value')
                obj_has_empty_string = 1
            elif type(v) is dict:
                check_empty_string_values(v)

