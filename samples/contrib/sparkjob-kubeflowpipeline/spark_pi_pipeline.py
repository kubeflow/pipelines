from kfp import dsl
from kfp import compiler

SPARK_YAML = """apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-pi
  namespace: default
spec:
  type: Scala
  mode: cluster
  image: docker.io/library/spark:4.0.0
  imagePullPolicy: IfNotPresent
  mainClass: org.apache.spark.examples.SparkPi
  mainApplicationFile: local:///opt/spark/examples/jars/spark-examples.jar
  arguments:
  - "5000"
  sparkVersion: 4.0.0
  timeToLiveSeconds: 600
  driver:
    labels:
      version: 4.0.0
    cores: 1
    memory: 512m
    serviceAccount: spark-operator-spark
    securityContext:
      capabilities:
        drop:
        - ALL
      runAsGroup: 185
      runAsUser: 185
      runAsNonRoot: true
      allowPrivilegeEscalation: false
      seccompProfile:
        type: RuntimeDefault
  executor:
    labels:
      version: 4.0.0
    instances: 1
    cores: 1
    memory: 512m
    securityContext:
      capabilities:
        drop:
        - ALL
      runAsGroup: 185
      runAsUser: 185
      runAsNonRoot: true
      allowPrivilegeEscalation: false
      seccompProfile:
        type: RuntimeDefault
"""

@dsl.component(
    base_image="vikassaxena02/vikas-kfpv2-python310-kubectl-nokfp-image:0.4"
)
def submit_spark_application(yaml_spec: str) -> str:
    import yaml
    import os
    import subprocess

    # Retrieve suffix from KFP_POD_NAME
    kfp_pod_name = os.environ.get("KFP_POD_NAME", "nopod")
    parts = kfp_pod_name.split("-")
    # IMPORTANT - adjust the value for suffix if you have no - in name field intemplate yaml or you have more than 1 - there 
    suffix = parts[5] if len(parts) >= 5 else "nosuffix"
    print(f"Using suffix derived from KFP_POD_NAME: {suffix}")

    # Load YAML into dict
    spec = yaml.safe_load(yaml_spec)

    # Append workflow UID to metadata.name
    spec["metadata"]["name"] += f"-{suffix}"

    # Write updated YAML back to file
    with open("/tmp/spark.yaml", "w") as f:
        yaml.dump(spec, f)

    # Use kubectl apply to create/update
    subprocess.run(["kubectl", "apply", "-f", "/tmp/spark.yaml"], check=True)

    # Return the actual SparkApplication name
    return spec["metadata"]["name"]

@dsl.component(
    base_image="vikassaxena02/vikas-kfpv2-python310-kubectl-nokfp-image:0.4"
)
def fetch_driver_logs(
    spark_app_name: str,
    spark_driver_logs: dsl.Output[dsl.Artifact],
):
    import subprocess
    import time
    import json

    # Wait for SparkApplication to complete
    print("Waiting for SparkApplication to complete...")
    for attempt in range(60):
        try:
            get_status_cmd = [
                "kubectl", "get", "sparkapplication", spark_app_name,
                "-n", "default", "-o", "json"
            ]
            output = subprocess.check_output(get_status_cmd, text=True)
            status_json = json.loads(output)
            app_state = status_json.get("status", {}).get("applicationState", {}).get("state", "")
            print(f"Attempt {attempt+1}: SparkApplication state: {app_state}")
            if app_state in ["COMPLETED", "FAILED"]:
                break
        except Exception as e:
            print("Error checking SparkApplication status:", str(e))
        time.sleep(10)
    else:
        raise RuntimeError("Timed out waiting for SparkApplication to complete.")

    # Now fetch the driver pod name
    pod_name = ""
    for i in range(6):
        try:
            pod_name_cmd = [
                "kubectl",
                "get",
                "pods",
                "-n", "default",
                "-l", f"spark-role=driver,spark-app-name={spark_app_name}",
                "-o", "jsonpath={.items[0].metadata.name}"
            ]
            pod_name = subprocess.check_output(pod_name_cmd, text=True).strip()
            if pod_name:
                print(f"Found driver pod: {pod_name}")
                break
        except subprocess.CalledProcessError as e:
            print(f"Attempt {i+1}: Failed to find driver pod, retrying...")
        time.sleep(5)
    else:
        raise RuntimeError("Failed to locate driver pod for SparkApplication.")

    # Print the driver pod name
    print("Driver pod:", pod_name)

    # Get logs
    logs = subprocess.check_output(
        ["kubectl", "logs", "-n", "default", pod_name],
        text=True
    )

    # Write logs to artifact
    with open(spark_driver_logs.path, "w") as f:
        f.write(logs)

    print("Driver logs saved to:", spark_driver_logs.path)

@dsl.pipeline(
    name="Spark Pi Pipeline KFP v2",
    description="Submit SparkApplication and Fetch Driver Logs"
)
def spark_pi_pipeline():
    submit_task = submit_spark_application(yaml_spec=SPARK_YAML)
    submit_task.set_caching_options(False)

    fetch_driver_logs_task = fetch_driver_logs(spark_app_name=submit_task.output)
    fetch_driver_logs_task.set_caching_options(False)

if __name__ == "__main__":
    import kfp
    from kfp import compiler

    pipeline_file = "spark_pi_pipeline.yaml"
    compiler.Compiler().compile(
        pipeline_func=spark_pi_pipeline,
        package_path=pipeline_file
    )

