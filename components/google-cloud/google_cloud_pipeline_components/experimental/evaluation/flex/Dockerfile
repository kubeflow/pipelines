# A DataFlow Flex Templates compatible container for model evaluation.

# TODO: Restore when 3.8 is in the base image.
# FROM gcr.io/dataflow-templates-base/python3-template-launcher-base
FROM gcr.io/dataflow-templates-base/python3-template-launcher-base-staging:flex_templates_base_image_release_20201026_RC00

RUN python --version

WORKDIR .
ADD . / /

# https://beam.apache.org/documentation/sdks/python-pipeline-dependencies/
# setup.py must be in the root directory for find_packages to work.
#
ADD setup.py /setup.py

ENV FLEX_TEMPLATE_PYTHON_PY_FILE "/main.py"
ENV FLEX_TEMPLATE_PYTHON_PY_OPTIONS ""
ENV FLEX_TEMPLATE_PYTHON_SETUP_FILE "/setup.py"

# Global for reuse elsewhere.
#
ENV REQUIREMENTS_FILE "/requirements.txt"
RUN cat "${REQUIREMENTS_FILE}"
RUN pip install --upgrade pip
RUN pip install -U -r "${REQUIREMENTS_FILE}"
