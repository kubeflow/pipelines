FROM pytorch/pytorch:latest

RUN pip install pandas flask-cors Minio Pillow torchsummary
RUN mkdir /tmp/data
RUN curl -c ./cookie -Ls "https://drive.google.com/uc?export=download&id=0BxYys69jI14kYVM3aVhKS1VhRUk" > /dev/null && curl -Lb ./cookie "https://drive.google.com/uc?export=download&confirm=`awk '/download/ {print $NF}' ./cookie`&id=0BxYys69jI14kYVM3aVhKS1VhRUk" -o /tmp/data/UTKFace.tar.gz
RUN cd /tmp/data && tar -xzvf ./UTKFace.tar.gz 2>&1 > dummy.log && rm ./UTKFace.tar.gz
RUN apt-get update && apt-get install -y zip

ENV APP_HOME /app
COPY . $APP_HOME
WORKDIR $APP_HOME

RUN zip model.zip PyTorchModel.py

ENTRYPOINT ["python"]
CMD ["gender_classification_training.py"]
