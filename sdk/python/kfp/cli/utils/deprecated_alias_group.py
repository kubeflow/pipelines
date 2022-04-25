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

from typing import Dict, List, Tuple, Union

import click


def deprecated_alias_group_factory(deprecated_map: Dict[str, str]):
    """Closure that returns a class that implements the deprecated alias group.

    Args:
        deprecated_map (Dict[str, str]): Dictionary mapping old deprecated names to new names.

    Raises:
        click.UsageError: If neither command nor alias is found.

    Returns:
        _type_: _description_
    """

    class DeprecatedAliasGroup(click.Group):

        def __init__(self, *args, **kwargs) -> None:
            super().__init__(*args, **kwargs)
            self.deprecated_map = deprecated_map

        def get_command(self, ctx: click.Context,
                        cmd_name: str) -> click.Command:
            # using the correct name
            command = click.Group.get_command(self, ctx, cmd_name)
            if command is not None:
                return command

            # using the deprecated alias
            correct_name = self.deprecated_map.get(cmd_name)
            if correct_name is None:
                raise click.UsageError(f"Unrecognized command '{cmd_name}'.")

            command = click.Group.get_command(self, ctx, correct_name)
            click.echo(
                f"Warning: '{cmd_name}' is deprecated, use '{correct_name}' instead.",
                err=True)

            return command

        def resolve_command(
            self, ctx: click.Context, args: List[str]
        ) -> Tuple[Union[str, None], Union[click.Command, None], List[str]]:
            # always return the full command name
            _, cmd, args = super().resolve_command(ctx, args)
            return cmd.name, cmd, args  # type: ignore

    return DeprecatedAliasGroup
