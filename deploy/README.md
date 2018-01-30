
# Building the Docker container

    docker build --rm -t docker-pipelines .

# Local Airflow using SQLite and the Airflow SequentialExecutor

Start Airflow:

    docker run -d -u airflow -p 8080:8080 --entrypoint "script/entrypoint_quickstart.sh" docker-pipelines quickstart

See the Airflow UI at:

    http://localhost:8080/

Get the running container ID using:

    docker ps

Log into the container:

    docker exec -it <CONTAINER_ID> bash

Run an Airflow command with the correct environment:

    ./script/entrypoint_quickstart.sh airflow version


# Local Airflow using PostgreSQL and the Airflow LocalExecutor

Start PostgreSQL:

    docker run -d --name=postgres -p 5432:5432 -e POSTGRES_USER=airflow -e POSTGRES_PASSWORD=airflow -e POSTGRES_DB=airflow postgres:10.1-alpine

Start Airflow:

    docker run -d -u airflow --entrypoint "script/entrypoint_local.sh" --net=host -e POSTGRES_USER=airflow -e POSTGRES_PASSWORD=airflow -e POSTGRES_DB=airflow -e POSTGRES_PORT=5432 -e POSTGRES_HOST=localhost docker-pipelines local

See the Airflow UI at:

    http://localhost:8080/

Get the running container ID using:

    docker ps

Log into the container:

    docker exec -it <CONTAINER_ID> bash

Run an Airflow command with the correct environment:

    ./script/entrypoint_local.sh airflow version


# Common Issues

When looking at the Airflow UI, if you see an error of the form:

    "TypeError: Object of type 'bytes' is not JSON serializable"

Try clearing your browser cookies ([link](https://github.com/puckel/docker-airflow/issues/116))
