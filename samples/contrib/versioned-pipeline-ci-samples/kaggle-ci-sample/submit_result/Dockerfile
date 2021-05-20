FROM python:3.7
ENV KAGGLE_USERNAME=[YOUR KAGGLE USERNAME] \
    KAGGLE_KEY=[YOUR KAGGLE KEY]
COPY requirements.txt .
RUN python3 -m pip install -r \
    requirements.txt --quiet --no-cache-dir \
    && rm -f requirements.txt
COPY ./submit_result.py .
CMD ["python", "submit_result.py"]