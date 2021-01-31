FROM tensorflow/tensorflow:2.0.0-py3
COPY requirements.txt .
RUN pip3 install -r requirements.txt
COPY ./visualize.py .
CMD ["python", 'visualize.py']