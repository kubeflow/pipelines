FROM python:3.7
ENV KAGGLE_USERNAME=ludai0919 \
    KAGGLE_KEY=a03e318b84bbf3766f4654dfc6e72922
RUN pip install kaggle
RUN pip install google-cloud-storage
COPY ./download_data.py .
CMD ["python", "download_data.py"]