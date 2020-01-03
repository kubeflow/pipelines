# ml-metadata package depends on tensorflow package
FROM python:3.7
RUN python3 -m pip install kubernetes 'ml-metadata==0.15.2' pyyaml --upgrade --quiet
COPY components/license.sh components/third_party_licenses.csv /kfp/metadata_writer/
RUN mkdir /usr/licenses && /kfp/metadata_writer/license.sh /kfp/metadata_writer/third_party_licenses.csv /usr/licenses

COPY sdk/metadata_writer/metadata_writer.py /kfp/metadata_writer/metadata_writer.py
CMD python3 /kfp/metadata_writer/metadata_writer.py
