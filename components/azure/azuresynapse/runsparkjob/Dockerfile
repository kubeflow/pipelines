FROM mcr.microsoft.com/azure-cli
RUN pip install --upgrade pip &&\
    az extension add --name synapse &&\
    apk add --update jq
COPY src/ /src/
CMD bash