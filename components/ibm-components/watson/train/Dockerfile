FROM python:3.6-slim
  
# Directories for model codes and secrets
RUN mkdir /app
RUN mkdir /app/secrets

# Watson studio and machine learning python client
COPY requirements.txt .
RUN python3 -m pip install -r \
    requirements.txt --quiet --no-cache-dir \
    && rm -f requirements.txt

# Python functions with endpoints to Watson Machine Learning
COPY src/wml-train.py /app
