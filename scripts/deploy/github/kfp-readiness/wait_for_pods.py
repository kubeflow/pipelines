import logging
import time
import urllib3
import sys
from kubernetes import client, config

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

namespace = 'kubeflow'

config.load_kube_config()
v1 = client.CoreV1Api()


def get_pod_statuses():
    pods = v1.list_namespaced_pod(namespace=namespace)
    statuses = {}
    for pod in pods.items:
        pod_name = pod.metadata.name
        pod_status = pod.status.phase
        container_statuses = pod.status.container_statuses or []
        ready_containers = sum(1 for status in container_statuses if status.ready)
        total_containers = len(container_statuses)
        statuses[pod_name] = (pod_status, ready_containers, total_containers)
    return statuses


def all_pods_ready(statuses):
    return all(pod_status == 'Running' and ready == total
               for pod_status, ready, total in statuses.values())


def check_pods(calm_time=10, timeout=600, retries_after_ready=5):
    start_time = time.time()
    stable_count = 0
    previous_statuses = {}

    while time.time() - start_time < timeout:
        current_statuses = get_pod_statuses()

        logging.info("Checking pod statuses...")
        for pod_name, (pod_status, ready, total) in current_statuses.items():
            logging.info(f"Pod {pod_name} - Status: {pod_status}, Ready: {ready}/{total}")

        if current_statuses == previous_statuses:
            if all_pods_ready(current_statuses):
                stable_count += 1
                if stable_count >= retries_after_ready:
                    logging.info("All pods are calm and fully ready.")
                    break
                else:
                    logging.info(
                        f"Pods are calm but have only been stable for {stable_count}/{retries_after_ready} retries.")
            else:
                stable_count = 0
        else:
            stable_count = 0

        previous_statuses = current_statuses
        logging.info(f"Pods are still stabilizing. Retrying in {calm_time} seconds...")
        time.sleep(calm_time)
    else:
        raise Exception("Pods did not stabilize within the timeout period.")

    logging.info("Final pod statuses:")
    for pod_name, (pod_status, ready, total) in previous_statuses.items():
        if pod_status == 'Running' and ready == total:
            logging.info(f"Pod {pod_name} is fully ready ({ready}/{total})")
        else:
            logging.info(f"Pod {pod_name} is not ready (Status: {pod_status}, Ready: {ready}/{total})")


if __name__ == "__main__":
    check_pods()
