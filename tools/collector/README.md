# Spartakus Collector In App Engine

This directory contains configs for deploying the [Spartakus](https://github.com/kubernetes-incubator/spartakus)
[Collector](https://github.com/kubernetes-incubator/spartakus/blob/master/cmd/spartakus/collector.go) to App Engine.

The collector is configured to write reports to a BigQuery table.

To install this collector in your app, run the following:

```sh
export PROJECT="<PROJECT_HOLDING_THE_APP_ENGINE_APP>"
export BQ_PROJECT="<PROJECT_HOLDING_THE_BQ_DATASET>"
export DATASET="spartakus_collector"
export TABLE="reports"

# Create the dataset for storing spartakus reports
bq --project="${BQ_PROJECT}" mk "${DATASET}"

# Download the schema for the reports table
curl https://raw.githubusercontent.com/kubernetes-incubator/spartakus/master/pkg/database/bigquery.schema.json -o spartakus.schema.json
# Create the reports table
bq --project="${BQ_PROJECT}" mk "${DATASET}.${TABLE}" ./spartakus.schema.json 

# Deploy the app
sed -i -e "s/__PROJECT_ID__/${BQ_PROJECT}/g" -e "s/__DATASET__/${DATASET}/g" -e "s/__TABLE__/${TABLE}/g" ./app.yaml
gcloud --project="${PROJECT}" app deploy ./

# Revert the unsaved changes to `app.yaml`
git checkout -- ./app.yaml
```
