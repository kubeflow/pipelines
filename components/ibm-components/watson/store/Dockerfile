FROM python:3.6-slim
  
# Directories for model codes and secrets
RUN mkdir /app
RUN mkdir /app/secrets

# Watson studio and machine learning python client
RUN pip install watson-machine-learning-client-V4>=1.0.110 minio

# Python functions with endpoints to Watson Machine Learning
COPY src/wml-store.py /app
