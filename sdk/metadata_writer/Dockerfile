# ml-metadata package depends on tensorflow package
FROM tensorflow/tensorflow:1.15.0-py3
RUN python3 -m pip install kubernetes 'ml-metadata==0.15.2' pyyaml --upgrade --quiet
COPY metadata_writer.py /kfp/metadata_writer/metadata_writer.py
CMD python3 /kfp/metadata_writer/metadata_writer.py
