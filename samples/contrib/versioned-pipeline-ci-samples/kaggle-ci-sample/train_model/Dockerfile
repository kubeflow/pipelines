FROM python:3.7
COPY ./train.py .
RUN pip install gcsfs
RUN pip install pandas numpy matplotlib seaborn sklearn
CMD ["python", "train.py"]