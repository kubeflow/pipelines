FROM python:3.6-slim
  
# Directories for model codes and secrets
RUN mkdir /app
RUN mkdir /app/secrets

# Watson studio and machine learning python client
COPY requirements.txt .
RUN pip3 install -r requirements.txt

# Python functions with endpoints to Watson Machine Learning
COPY src/wml-train.py /app
