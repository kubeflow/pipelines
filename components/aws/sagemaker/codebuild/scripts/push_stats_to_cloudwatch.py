import boto3
from datetime import datetime
import xml.etree.ElementTree as ET
import os


xml_path = "./integration_tests.xml"


def readXML_and_publish_metrics_to_cw():
    if os.path.isfile(xml_path):
        tree = ET.parse(xml_path)
        testsuite = tree.find("testsuite")
        failures = testsuite.attrib["failures"]
        tests = testsuite.attrib["tests"]
        successes = int(tests) - int(failures)
        success_rate = (successes/int(tests))*100
    else:
        failures = 0
        successes = 0
        tests = 0
        success_rate = 0

    timestamp = datetime.now().strftime("%Y-%m-%dT%H:%M:%S")

    print(f"Failures: {failures}")
    print(f"Total tests: {tests}")
    print(f"Success: {successes}")

    # push to cloudwatch
    cw_client = boto3.client("cloudwatch")
    project_name = os.getenv("PROJECT_NAME")

    # Define the metric data
    metric_data = [
        {
            "MetricName": "failures",
            "Timestamp": timestamp,
            "Dimensions": [
                {"Name": "CodeBuild Project Name", "Value": project_name},
            ],
            "Value": int(failures),
            "Unit": "Count",
        },
        {
            "MetricName": "total_tests",
            "Timestamp": timestamp,
            "Dimensions": [
                {"Name": "CodeBuild Project Name", "Value": project_name},
            ],
            "Value": int(tests),
            "Unit": "Count",
        },
        {
            "MetricName": "successes",
            "Timestamp": timestamp,
            "Dimensions": [
                {"Name": "CodeBuild Project Name", "Value": project_name},
            ],
            "Value": int(successes),
            "Unit": "Count",
        },
        {
            "MetricName": "success_rate",
            "Timestamp": timestamp,
            "Dimensions": [
                {"Name": "CodeBuild Project Name", "Value": project_name},
            ],
            "Value": int(success_rate),
            "Unit": "Percent",
        },
    ]

    # Use the put_metric_data method to push the metric data to CloudWatch
    try:
        response = cw_client.put_metric_data(
            Namespace="Canary_Metrics", MetricData=metric_data
        )
        if response["ResponseMetadata"]["HTTPStatusCode"] == 200:
            print("Successfully pushed data to CloudWatch")
            # return 200 status code if successful
            return 200
        else:
            # raise exception if the status code is not 200
            raise Exception(
                "Unexpected response status code: {}".format(
                    response["ResponseMetadata"]["HTTPStatusCode"]
                )
            )
    except Exception as e:
        print("Error pushing data to CloudWatch: {}".format(e))
        # raise exception if there was an error pushing data to CloudWatch
        raise


def main():
    readXML_and_publish_metrics_to_cw()


if __name__ == "__main__":
    main()