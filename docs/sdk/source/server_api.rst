kfp.server_api
==============

The generated low-level REST API client is included in ``kfp``. Most users
should use :class:`kfp.Client`; direct users of the former ``kfp_server_api``
module should update their imports::

   from kfp.server_api import ApiClient, Configuration, RunServiceApi

The client is regenerated from the backend's OpenAPI specification. It does
not implement a server.
