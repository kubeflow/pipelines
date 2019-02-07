FROM ml-pipeline-local-base

RUN pip install numpy==1.15.0 \
        tensorflow==1.11.0 \
        tensorboard==1.11.0

WORKDIR /ml

EXPOSE 6006

ENTRYPOINT [ "tensorboard" ]