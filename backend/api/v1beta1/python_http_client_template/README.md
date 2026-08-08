This folder contains overrides to openapi-generator python http client templates.

`setup.mustache` generates a PEP 621 `pyproject.toml` (written as `setup.py` by
the generator, then renamed in `build_kfp_server_api_python_package.sh`).
`requirements.mustache` and `setup_cfg.mustache` are intentionally empty so the
committed tree only contains `pyproject.toml`.

Resources:
* Documentation for overriding templates: https://github.com/OpenAPITools/openapi-generator/tree/v4.3.1/modules/openapi-generator/src/main/resources/python.
* Original templates for the generator version we use: https://github.com/OpenAPITools/openapi-generator/tree/v4.3.1/modules/openapi-generator/src/main/resources/python
