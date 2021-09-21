#!/usr/bin/env python3

import json
from kfp import components

component_store = components.ComponentStore.default_store
component_store._refresh_component_cache()
component_urls = component_store._url_to_info_db.keys()

def make_sortable_component_url(url: str) -> str:
    if url.startswith('https://raw.githubusercontent.com'):
        url_parts = url.split('/')
        # Set commit/branch ref to master, so that the URLs can be sorted
        url_parts[5] = 'master'
        return '/'.join(url_parts).casefold()
    return url

component_urls = sorted(component_urls, key=make_sortable_component_url)

component_list_lines = []
for component_url in component_urls:
    component_info = json.loads(component_store._url_to_info_db.try_get_value_bytes(component_url))
    component_name = component_info['name']
    url_parts = component_url.split('/')

    repo_org = url_parts[3]
    repo_name = url_parts[4]
    commit_ref = url_parts[5]
    path_parts = url_parts[6:]

    parts_to_skip = 0

    if path_parts[0] == 'components':
        parts_to_skip += 1
    else:
        # Only list components from the "/components/" directory for now
        continue

    component_path_line = ""
    for part_idx in range(parts_to_skip, len(path_parts) - 1):
        path_part = path_parts[part_idx]
        part_url = f'https://github.com/{repo_org}/{repo_name}/tree/{commit_ref}/' + '/'.join(path_parts[0:part_idx + 1])
        component_path_line += f' / [{path_part}]({part_url})'

    component_path_line += f' / [{component_name}]({component_url})'
    component_list_lines.append(component_path_line)

# Updating the README
with open('README.md', 'r') as f:
    readme_lines = f.readlines()

with open('README.md', 'w') as f:
    for line in readme_lines:
        f.write(line)
        if 'Index of components' in line:
            break
    for line in component_list_lines:
        f.write(line + '\n\n')
