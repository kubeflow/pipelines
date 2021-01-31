FROM python:3.7
ENV KAGGLE_USERNAME=[YOUR KAGGLE USERNAME] \
    KAGGLE_KEY=[YOUR KAGGLE KEY]
COPY requirements.txt .
RUN pip3 install -r requirements.txt
COPY ./submit_result.py .
CMD ["python", "submit_result.py"]