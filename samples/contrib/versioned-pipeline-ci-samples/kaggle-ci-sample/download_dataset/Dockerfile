FROM python:3.7
ENV KAGGLE_USERNAME=[your kaggle_username] \
    KAGGLE_KEY=[your kaggle_key]
RUN pip install kaggle
RUN pip install google-cloud-storage
COPY ./download_data.py .
CMD ["python", "download_data.py"]