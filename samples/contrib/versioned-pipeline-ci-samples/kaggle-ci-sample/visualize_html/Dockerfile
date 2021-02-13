FROM tensorflow/tensorflow:2.0.0-py3
COPY requirements.txt .
RUN python3 -m pip install -r \
    requirements.txt --quiet --no-cache-dir \
    && rm -f requirements.txt
COPY ./visualize.py .
CMD ["python", 'visualize.py']