# Copyright 2022 The Kubeflow Authors
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

from typing import List, Optional, Tuple

import click


class AliasedPluralsGroup(click.Group):

    def get_command(self, ctx: click.Context, cmd_name: str) -> click.Command:
        regular = click.Group.get_command(self, ctx, cmd_name)
        if regular is not None:
            return regular
        elif cmd_name.endswith('s'):
            singular = click.Group.get_command(self, ctx, cmd_name[:-1])
            if singular is not None:
                return singular
        raise click.UsageError(f"Unrecognized command '{cmd_name}'")

    def resolve_command(
            self, ctx: click.Context,
            args: List[str]) -> Tuple[Optional[str], Optional[None], List[str]]:
        # always return the full command name
        _, cmd, args = super().resolve_command(ctx, args)
        return cmd.name, cmd, args  # type: ignore
