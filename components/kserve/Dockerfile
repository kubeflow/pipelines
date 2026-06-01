FROM python:3.11-slim-bullseye
RUN apt-get update && apt-get install -y gcc python3-dev

RUN python3 -m pip install uv \
    && uv pip install --system \
      "kserve==0.12.0" \
      "protobuf~=3.19.0" \
      --quiet --no-cache-dir

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

ENTRYPOINT ["python"]
CMD ["kservedeployer.py"]
