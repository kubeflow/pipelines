from typing import Union

from kfp.components import placeholders


class ContainerComponentArtifactChannel():
    """A class for passing in placeholders into container_component decorated
    function."""

    def __init__(self, io_type: str, var_name: str):
        self._io_type = io_type
        self._var_name = var_name

    def __getattr__(
        self, _name: str
    ) -> Union[placeholders.InputUriPlaceholder, placeholders
               .InputPathPlaceholder, placeholders.OutputUriPlaceholder,
               placeholders.OutputPathPlaceholder]:
        if _name not in ['uri', 'path']:
            raise AttributeError(f'Cannot access artifact attribute "{_name}".')
        if self._io_type == 'input':
            if _name == 'uri':
                return placeholders.InputUriPlaceholder(self._var_name)
            elif _name == 'path':
                return placeholders.InputPathPlaceholder(self._var_name)
        elif self._io_type == 'output':
            if _name == 'uri':
                return placeholders.OutputUriPlaceholder(self._var_name)
            elif _name == 'path':
                return placeholders.OutputPathPlaceholder(self._var_name)
