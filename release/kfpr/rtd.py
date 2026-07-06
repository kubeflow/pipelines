"""Read the Docs API helpers for release automation."""

import json
import time
import urllib.error
import urllib.request
from collections.abc import Callable
from typing import Any

API_BASE = 'https://app.readthedocs.org/api/v3'


class ReadTheDocsError(RuntimeError):
  """Raised when Read the Docs automation cannot complete."""

  def __init__(self, message: str, status_code: int | None = None):
    super().__init__(message)
    self.status_code = status_code


class ReadTheDocsClient:
  """Tiny Read the Docs API v3 client."""

  def __init__(self, token: str, base_url: str = API_BASE):
    self.token = token
    self.base_url = base_url.rstrip('/')

  def request(
      self,
      method: str,
      path: str,
      body: dict[str, object] | None = None,
      expected: tuple[int, ...] = (200,),
  ) -> Any:
    data = None if body is None else json.dumps(body).encode()
    request = urllib.request.Request(
        f'{self.base_url}{path}',
        data=data,
        method=method,
        headers={
            'Authorization': f'Token {self.token}',
            'Content-Type': 'application/json',
        },
    )
    try:
      response = urllib.request.urlopen(request, timeout=30)
    except urllib.error.HTTPError as error:
      detail = error.read().decode(errors='replace')
      raise ReadTheDocsError(
          f'RTD API {method} {path} failed: HTTP {error.code} {detail}',
          status_code=error.code,
      ) from error
    except urllib.error.URLError as error:
      raise ReadTheDocsError(f'RTD API {method} {path} failed: {error.reason}') from error
    if response.status not in expected:
      raise ReadTheDocsError(f'RTD API {method} {path} returned HTTP {response.status}')
    payload = response.read()
    if not payload:
      return None
    return json.loads(payload)

  def sync_versions(self, project: str) -> None:
    self.request('POST', f'/projects/{project}/sync-versions/', expected=(202,))

  def activate_version(self, project: str, version: str) -> None:
    self.request('PATCH', f'/projects/{project}/versions/{version}/', {'active': True}, expected=(204,))

  def get_version(self, project: str, version: str) -> dict[str, Any]:
    return self.request('GET', f'/projects/{project}/versions/{version}/')

  def trigger_build(self, project: str, version: str) -> int:
    data = self.request('POST', f'/projects/{project}/versions/{version}/builds/', expected=(202,))
    build = data['build']
    return int(build['id'] if isinstance(build, dict) else build)

  def get_build(self, project: str, build_id: int) -> dict[str, Any]:
    return self.request('GET', f'/projects/{project}/builds/{build_id}/')

  def patch_project(self, project: str, body: dict[str, object]) -> None:
    self.request('PATCH', f'/projects/{project}/', body, expected=(204,))


def wait_for_build(
    client: ReadTheDocsClient,
    project: str,
    build_id: int,
    sleep: Callable[[float], None] = time.sleep,
) -> None:
  for _ in range(60):
    build = client.get_build(project, build_id)
    if build.get('state', {}).get('code') == 'finished':
      if build.get('success'):
        return
      raise ReadTheDocsError(str(build.get('error') or f'{project} build {build_id} failed'))
    sleep(10)
  raise ReadTheDocsError(f'{project} build {build_id} did not finish in time')


def wait_for_version(
    client: ReadTheDocsClient,
    project: str,
    version: str,
    sleep: Callable[[float], None] = time.sleep,
) -> None:
  for _ in range(12):
    try:
      client.get_version(project, version)
      return
    except ReadTheDocsError as error:
      if error.status_code != 404:
        raise
      sleep(5)
  raise ReadTheDocsError(f'{project} version {version} did not sync in time')


def update_release_docs(
    client: ReadTheDocsClient,
    version: str,
    release_branch: str,
    sleep: Callable[[float], None] = time.sleep,
) -> None:
  projects = [
      (
          'kubeflow-pipelines',
          f'sdk-{version}',
          {'default_version': f'sdk-{version}', 'default_branch': release_branch},
      ),
      (
          'kfp-kubernetes',
          f'kfp-kubernetes-{version}',
          {'default_version': f'kfp-kubernetes-{version}'},
      ),
  ]
  for project, _, _ in projects:
    client.sync_versions(project)
  for project, version_slug, _ in projects:
    wait_for_version(client, project, version_slug, sleep=sleep)
  for project, version_slug, _ in projects:
    client.activate_version(project, version_slug)
  for project, version_slug, _ in projects:
    build_id = client.trigger_build(project, version_slug)
    wait_for_build(client, project, build_id, sleep=sleep)
  for project, _, defaults in projects:
    client.patch_project(project, defaults)
