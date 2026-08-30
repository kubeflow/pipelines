# Copyright 2026 The Kubeflow Authors
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
"""Structure-aware metadata discovery for repository Go version policy."""

from pathlib import Path
import re
from typing import Dict, Iterable, List, Optional, Tuple

GO_DOWNLOAD_PATTERN = re.compile(r'(?:dl\.google\.com/go/|go\.dev/dl/)go',
                                 re.IGNORECASE)
GO_TEXT_REFERENCE_PATTERN = re.compile(
    r'(?:\bgolang(?=[:@])|'
    r'^[ \t]*FROM(?:[ \t]+--platform=\S+)?[ \t]+(?:\S+/)?golang(?=[ \t]|$)|'
    r'(?:dl\.google\.com/go/|go\.dev/dl/)go)',
    re.IGNORECASE | re.MULTILINE,
)
MODULE_BLOCK_START_PATTERN = re.compile(
    r'^[ \t]*[A-Za-z][A-Za-z0-9]*[ \t]*\([ \t]*(?://.*)?$')
MODULE_BLOCK_END_PATTERN = re.compile(r'^[ \t]*\)[ \t]*(?://.*)?$')
DOCKER_ARG_PATTERN = re.compile(
    r'^[ \t]*ARG[ \t]+(?P<name>[A-Za-z_][A-Za-z0-9_]*)'
    r'(?:=(?P<value>.*))?$', re.IGNORECASE)
DOCKER_FROM_PATTERN = re.compile(
    r'^[ \t]*FROM(?:[ \t]+--platform=\S+)?[ \t]+'
    r'(?P<source>\S+)', re.IGNORECASE)
DOCKER_VARIABLE_PATTERN = re.compile(
    r'\$\{(?P<braced>[A-Za-z_][A-Za-z0-9_]*)\}|'
    r'\$(?P<plain>[A-Za-z_][A-Za-z0-9_]*)')


def top_level_module_matches(contents: str,
                             pattern: re.Pattern) -> List[re.Match]:
    """Return directive matches outside parenthesized go.mod blocks."""
    matches = []
    in_block = False
    for line in contents.splitlines():
        if in_block:
            if MODULE_BLOCK_END_PATTERN.fullmatch(line):
                in_block = False
            continue
        if MODULE_BLOCK_START_PATTERN.fullmatch(line):
            in_block = True
            continue
        match = pattern.fullmatch(line)
        if match is not None:
            matches.append(match)
    return matches


def yaml_mapping_values(contents: str,
                        target_keys: Iterable[str]) -> Dict[str, List[str]]:
    """Read scalar values for selected YAML mapping keys.

    This intentionally implements only YAML structure needed for metadata
    discovery: block mappings, flow mappings/sequences, quoted scalars,
    comments, and block scalars. It does not deserialize or execute YAML.
    """
    targets = set(target_keys)
    found = {key: [] for key in targets}
    block_scalar_parent_indent: Optional[int] = None

    for original_line in contents.splitlines():
        indent = len(original_line) - len(original_line.lstrip(' '))
        if block_scalar_parent_indent is not None:
            if not original_line.strip() or indent > block_scalar_parent_indent:
                continue
            block_scalar_parent_indent = None

        line = _strip_yaml_comment(original_line)
        body = line.lstrip(' ')
        if not body:
            continue
        if body.startswith('-') and (len(body) == 1 or body[1].isspace() or
                                     body[1] in '{['):
            body = body[1:].lstrip()

        entry = _mapping_entry(body, 0)
        if entry is None:
            if body.startswith(('{', '[')):
                _scan_flow(body, 0, targets, found)
            continue
        key, value_start = entry
        value_text = body[value_start:].lstrip()
        if value_text.startswith(('|', '>')):
            block_scalar_parent_indent = indent
            continue
        if value_text.startswith(('{', '[')):
            _scan_flow(value_text, 0, targets, found)
        elif key in targets:
            value, _ = _scalar(value_text, 0, flow=False)
            if value is not None:
                found[key].append(value)
    return found


def has_go_runtime_reference(relative_path: Path, contents: str) -> bool:
    if relative_path.suffix.lower() in {'.yaml', '.yml'}:
        values = yaml_mapping_values(contents, ('container', 'image'))
        if any(_is_golang_image(value)
               for values_for_key in values.values()
               for value in values_for_key):
            return True
        active_text = '\n'.join(
            _strip_yaml_comment(line) for line in contents.splitlines())
        return GO_DOWNLOAD_PATTERN.search(active_text) is not None

    if relative_path.name.startswith('Dockerfile'):
        stages, repository_arguments = docker_go_runtime_sources(contents)
        if stages or repository_arguments:
            return True

    active_text = '\n'.join(
        line for line in contents.splitlines()
        if not line.lstrip().startswith('#'))
    return GO_TEXT_REFERENCE_PATTERN.search(active_text) is not None


def docker_go_runtime_sources(contents: str) -> Tuple[List[str], List[str]]:
    """Return Golang FROM sources and literal global ARG defaults."""
    global_arguments = {}
    go_repository_arguments = []
    go_stages = []
    seen_from = False

    for line in _docker_logical_lines(contents):
        stripped = line.strip()
        if not stripped or stripped.startswith('#'):
            continue
        argument = DOCKER_ARG_PATTERN.fullmatch(line)
        if argument is not None and not seen_from:
            value = argument.group('value')
            if value is not None:
                value = value.strip().strip('"\'')
                global_arguments[argument.group('name')] = value
                if _is_golang_image(value):
                    go_repository_arguments.append(argument.group('name'))
            continue

        from_instruction = DOCKER_FROM_PATTERN.match(line)
        if from_instruction is None:
            continue
        seen_from = True
        source = from_instruction.group('source')
        resolved = DOCKER_VARIABLE_PATTERN.sub(
            lambda match: global_arguments.get(
                match.group('braced') or match.group('plain'), match.group(0)),
            source,
        )
        if _is_golang_image(resolved):
            go_stages.append(source)
    return go_stages, go_repository_arguments


def has_setup_go_use(contents: str) -> bool:
    return any(
        value.startswith('actions/setup-go@')
        for value in yaml_mapping_values(contents, ('uses',))['uses'])


def _is_golang_image(value: str) -> bool:
    image = value.strip()
    if image.startswith('docker://'):
        image = image[len('docker://'):]
    name = image.rsplit('/', 1)[-1]
    return re.match(r'^golang(?=[:@]|$)', name, re.IGNORECASE) is not None


def _docker_logical_lines(contents: str) -> List[str]:
    logical_lines = []
    pending = ''
    for line in contents.splitlines():
        current = pending + line.lstrip() if pending else line
        if current.rstrip().endswith('\\'):
            pending = current.rstrip()[:-1] + ' '
            continue
        logical_lines.append(current)
        pending = ''
    if pending:
        logical_lines.append(pending)
    return logical_lines


def _strip_yaml_comment(line: str) -> str:
    quote = None
    escaped = False
    index = 0
    while index < len(line):
        char = line[index]
        if quote == '"':
            if escaped:
                escaped = False
            elif char == '\\':
                escaped = True
            elif char == quote:
                quote = None
        elif quote == "'":
            if char == quote:
                if index + 1 < len(line) and line[index + 1] == quote:
                    index += 1
                else:
                    quote = None
        elif char in "'\"":
            quote = char
        elif char == '#' and (index == 0 or line[index - 1].isspace()):
            return line[:index]
        index += 1
    return line


def _mapping_entry(text: str, position: int) -> Optional[Tuple[str, int]]:
    key, position = _scalar(text, position, flow=True, key=True)
    if key is None:
        return None
    position = _skip_space(text, position)
    if position >= len(text) or text[position] != ':':
        return None
    return key, position + 1


def _scan_flow(text: str, position: int, targets: set,
               found: Dict[str, List[str]]) -> int:
    opener = text[position]
    closer = '}' if opener == '{' else ']'
    position += 1
    while position < len(text):
        position = _skip_space_and_commas(text, position)
        if position >= len(text) or text[position] == closer:
            return min(position + 1, len(text))
        if text[position] in '}]':
            return position + 1
        if text[position] in '{[':
            position = _scan_flow(text, position, targets, found)
            continue

        entry = _mapping_entry(text, position)
        if entry is None:
            _, next_position = _scalar(text, position, flow=True)
            position = max(position + 1, next_position)
            continue
        key, position = entry
        position = _skip_space(text, position)
        if position < len(text) and text[position] in '{[':
            position = _scan_flow(text, position, targets, found)
            continue
        value, position = _scalar(text, position, flow=True)
        if key in targets and value is not None:
            found[key].append(value)
    return position


def _scalar(text: str,
            position: int,
            flow: bool,
            key: bool = False) -> Tuple[Optional[str], int]:
    position = _skip_space(text, position)
    if position >= len(text):
        return None, position
    if text[position] in "'\"":
        return _quoted_scalar(text, position)

    start = position
    while position < len(text):
        char = text[position]
        if char in '{}[],' or (key and char == ':'):
            break
        if not flow and char.isspace():
            break
        position += 1
    value = text[start:position].strip()
    return (value if value else None), position


def _quoted_scalar(text: str, position: int) -> Tuple[str, int]:
    quote = text[position]
    position += 1
    value = []
    while position < len(text):
        char = text[position]
        if quote == "'" and char == quote and position + 1 < len(text) and \
                text[position + 1] == quote:
            value.append(quote)
            position += 2
            continue
        if char == quote:
            return ''.join(value), position + 1
        if quote == '"' and char == '\\' and position + 1 < len(text):
            position += 1
            value.append(text[position])
        else:
            value.append(char)
        position += 1
    return ''.join(value), position


def _skip_space(text: str, position: int) -> int:
    while position < len(text) and text[position].isspace():
        position += 1
    return position


def _skip_space_and_commas(text: str, position: int) -> int:
    while position < len(text) and (text[position].isspace() or
                                    text[position] == ','):
        position += 1
    return position
