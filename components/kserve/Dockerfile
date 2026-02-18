FROM python:3.11-slim-bullseye
RUN apt-get update && apt-get install -y gcc python3-dev

COPY requirements.txt .
RUN python3 -m pip install -r \
    requirements.txt --quiet --no-cache-dir \
    && rm -f requirements.txt

ENV APP_HOME /app
COPY src $APP_HOME
WORKDIR $APP_HOME

# Run as non-root user for security hardening
RUN adduser --system --uid 1000 appuser
USER appuser

ENTRYPOINT ["python"]
CMD ["kservedeployer.py"]
