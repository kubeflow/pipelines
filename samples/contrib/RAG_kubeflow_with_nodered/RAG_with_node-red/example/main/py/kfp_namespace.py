import os
import requests

from typing import TypedDict

def retrieve_namespaces(host: str, auth_session: TypedDict) -> str:
    workgroup_endpoint = os.path.join(host, "api/workgroup/env-info")

    cookies = {}
    cookie_tokens = auth_session["session_cookie"].split("=")
    print(cookie_tokens[0])
    cookies[cookie_tokens[0]]=cookie_tokens[1]
    resp = requests.get(workgroup_endpoint, cookies=cookies)
    if resp.status_code != 200:
        raise RuntimeError(
            f"HTTP status code '{resp.status_code}' for GET against: {workgroup_endpoint}"
        )
    return [ns["namespace"] for ns in resp.json()["namespaces"] if
            ns["role"]=="owner"]