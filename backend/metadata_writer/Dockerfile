# ml-metadata package depends on tensorflow package
FROM python:3.7
RUN python3 -m pip install 'kubernetes>=8.0.0,<11.0.0' 'ml-metadata==0.15.2' pyyaml --upgrade --quiet
COPY components/license.sh components/third_party_licenses.csv /kfp/metadata_writer/
RUN mkdir /usr/licenses && /kfp/metadata_writer/license.sh /kfp/metadata_writer/third_party_licenses.csv /usr/licenses

COPY backend/metadata_writer/src/* /kfp/metadata_writer/
CMD python3 -u /kfp/metadata_writer/metadata_writer.py
