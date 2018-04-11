FROM gcr.io/google_containers/spartakus-amd64:v1.0.0

EXPOSE 8080

ENV PROJECT "__PROJECT_ID__"
ENV DATASET "spartakus_collector"
ENV TABLE "reports"

CMD ["collector", "--database", "bigquery://${PROJECT}.${DATASET}.${TABLE}", "--v", "10"]
