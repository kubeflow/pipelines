FROM python:3.9-slim-bullseye
RUN apt-get update && apt-get install -y gcc python3-dev

COPY requirements.txt .
RUN pip install --upgrade pip
RUN python3 -m pip install --upgrade -r \
    requirements.txt --quiet --no-cache-dir \
    && rm -f requirements.txt

ENV APP_HOME /app
COPY kservedeployer.py $APP_HOME/kservedeployer.py
WORKDIR $APP_HOME

ENTRYPOINT ["python"]
CMD ["kservedeployer.py"]
