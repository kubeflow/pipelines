FROM python:3.7
COPY ./train.py .
COPY requirements.txt .
RUN pip3 install -r requirements.txt
CMD ["python", "train.py"]