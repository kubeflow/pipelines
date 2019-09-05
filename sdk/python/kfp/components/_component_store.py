__all__ = [
    'ComponentStore',
]

from pathlib import Path
import requests
from typing import Callable
from . import _components as comp
from ._structures import ComponentReference

class ComponentStore:
    def __init__(self, local_search_paths=None, url_search_prefixes=None):
        self.local_search_paths = local_search_paths or ['.']
        self.url_search_prefixes = url_search_prefixes or []

        self._component_file_name = 'component.yaml'
        self._digests_subpath = 'versions/sha256'
        self._tags_subpath = 'versions/tags'

    def load_component_from_url(self, url):
        return comp.load_component_from_url(url)

    def load_component_from_file(self, path):
        return comp.load_component_from_file(path)

    def load_component(self, name, digest=None, tag=None):
        '''
        Loads component local file or URL and creates a task factory function

        Search locations:
        <local-search-path>/<name>/component.yaml
        <url-search-prefix>/<name>/component.yaml

        If the digest is specified, then the search locations are:
        <local-search-path>/<name>/versions/sha256/<digest>
        <url-search-prefix>/<name>/versions/sha256/<digest>

        If the tag is specified, then the search locations are:
        <local-search-path>/<name>/versions/tags/<digest>
        <url-search-prefix>/<name>/versions/tags/<digest>

        Args:
            name:   Component name used to search and load the component artifact containing the component definition.
                    Component name usually has the following form: group/subgroup/component
            digest: Strict component version. SHA256 hash digest of the component artifact file. Can be used to load a specific component version so that the pipeline is reproducible.
            tag:    Version tag. Can be used to load component version from a specific branch. The version of the component referenced by a tag can change in future.

        Returns:
            A factory function with a strongly-typed signature.
            Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp).
        '''
        #This function should be called load_task_factory since it returns a factory function.
        #The real load_component function should produce an object with component properties (e.g. name, description, inputs/outputs).
        #TODO: Change this function to return component spec object but it should be callable to construct tasks.
        if not name:
            raise TypeError("name is required")
        if name.startswith('/') or name.endswith('/'):
            raise ValueError('Component name should not start or end with slash: "{}"'.format(name))

        tried_locations = []

        if digest is not None and tag is not None:
            raise ValueError('Cannot specify both tag and digest')

        if digest is not None:
            path_suffix = name + '/' + self._digests_subpath + '/' + digest
        elif tag is not None:
            path_suffix = name + '/' + self._tags_subpath + '/' + tag
            #TODO: Handle symlinks in GIT URLs
        else:
            path_suffix = name + '/' + self._component_file_name

        #Trying local search paths
        for local_search_path in self.local_search_paths:
            component_path = Path(local_search_path, path_suffix)
            tried_locations.append(str(component_path))
            if component_path.is_file():
                component_ref = ComponentReference(name=name, digest=digest, tag=tag)
                return comp.load_component_from_file(str(component_path))

        #Trying URL prefixes
        for url_search_prefix in self.url_search_prefixes:
            url = url_search_prefix + path_suffix
            tried_locations.append(url)
            try:
                response = requests.get(url) #Does not throw exceptions on bad status, but throws on dead domains and malformed URLs. Should we log those cases?
                response.raise_for_status()
            except:
                continue
            if response.content:
                component_ref = ComponentReference(name=name, digest=digest, tag=tag, url=url)
                return comp._load_component_from_yaml_or_zip_bytes(response.content, url, component_ref)

        raise RuntimeError('Component {} was not found. Tried the following locations:\n{}'.format(name, '\n'.join(tried_locations)))

    def _load_component_from_ref(self, component_ref: ComponentReference) -> Callable:
        if component_ref.spec:
            return comp._create_task_factory_from_component_spec(component_spec=component_ref.spec, component_ref=component_ref)
        if component_ref.url:
            return self.load_component_from_url(component_ref.url)
        return self.load_component(
            name=component_ref.name,
            digest=component_ref.digest,
            tag=component_ref.tag,
        )


ComponentStore.default_store = ComponentStore(
    local_search_paths=[
        '.',
    ],
    url_search_prefixes=[
        'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/'
    ],
)
