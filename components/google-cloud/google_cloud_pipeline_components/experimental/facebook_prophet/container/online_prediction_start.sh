#!/bin/bash
cd "$(dirname "$0")"
gunicorn -b 0.0.0.0:8080 online_prediction:app
