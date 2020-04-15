# ml-metadata package depends on tensorflow package
FROM python:3.7
COPY backend/metadata_writer/requirements.txt /kfp/metadata_writer/
RUN python3 -m pip install -r /kfp/metadata_writer/requirements.txt
COPY components/license.sh components/third_party_licenses.csv /kfp/metadata_writer/
RUN mkdir /usr/licenses && /kfp/metadata_writer/license.sh /kfp/metadata_writer/third_party_licenses.csv /usr/licenses

COPY backend/metadata_writer/src/* /kfp/metadata_writer/
CMD python3 -u /kfp/metadata_writer/metadata_writer.py
