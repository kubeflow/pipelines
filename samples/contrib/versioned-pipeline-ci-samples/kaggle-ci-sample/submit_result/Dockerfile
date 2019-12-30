FROM python:3.7
ENV KAGGLE_USERNAME=[Your kaggle username] \
    KAGGLE_KEY=[Your kaggle key]
RUN pip install kaggle
RUN pip install gcsfs
COPY ./submit_result.py .
CMD ["python", "submit_result.py"]