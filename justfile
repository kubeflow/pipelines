set shell := ["sh", "-cu"]

default:
    @just --list

sdk-test:
    make sdk-test

frontend-test:
    make frontend-test

build:
    make build

generate-python:
    make generate-python

native-acceptance:
    bash test/native-deployment.sh
