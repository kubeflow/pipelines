import argparse
import datetime
import json
import os
import sys
import logging
import requests
import subprocess
import six
import time
import yaml
from subprocess import Popen,PIPE
from shlex import split

def setup_custom_logging():
  logging.basicConfig(format='%(asctime)s %(levelname)-8s %(message)s', 
                      level=logging.INFO,
                      datefmt='%Y-%m-%d %H:%M:%S')

def str2bool(v):
  return v.lower() in ("yes", "true", "t", "1")

def submit_job(command):
  logging.info("command: {0}".format(command))
  try:
    output = subprocess.check_output(command, stderr=subprocess.STDOUT, shell=True)
    result = output.decode()
  except subprocess.CalledProcessError as exc:
    print("Status : FAIL", exc.returncode, exc.output)
    sys.exit(-1)
  logging.info('Submit Job: %s.' % result)

def _is_active_status(status):
    logging.info("status: {0}".format(status))
    return status == 'PENDING' or status == 'RUNNING'

def _is_pending_status(status):
    logging.info("status: {0}".format(status))
    return status == 'PENDING'

def wait_job_done(name, job_type, timeout):
  end_time = datetime.datetime.now() + timeout
  logging.info("expect done time: {0}".format(end_time))
  status = get_job_status(name, job_type)
  while _is_active_status(status):
    if datetime.datetime.now() > end_time:
      timeoutMsg = "Timeout waiting for job {0} with job type {1} completing.".format(name ,job_type)
      logging.error(timeoutMsg)
      raise Exception(timeoutMsg)
    time.sleep(3)
    status = get_job_status(name, job_type)
  logging.info("job {0} with type {1} status is {2}".format(name, job_type, status))

def wait_job_running(name, job_type, timeout):
  end_time = datetime.datetime.now() + timeout
  logging.info("expect running time: {0}".format(end_time))
  status = get_job_status(name, job_type)
  while _is_pending_status(status):
    if datetime.datetime.now() > end_time:
      timeoutMsg = "Timeout waiting for job {0} with job type {1} running.".format(name ,job_type)
      logging.error(timeoutMsg)
      raise Exception(timeoutMsg)
    time.sleep(10)
    status = get_job_status(name, job_type)
  logging.info("job {0} with type {1} status is {2}".format(name, job_type, status))

def job_logging(name, job_type):
  logging_cmd = "arena logs -f %s" % (name)
  process = Popen(split(logging_cmd), stdout = PIPE, stderr = PIPE, encoding='utf8')
  while True:
    output = process.stdout.readline()
    if output == "" and process.poll() is not None:
      break
    if output:
      # print("", output.strip())
      logging.info(output.strip())
  rc = process.poll()
  return rc

def collect_metrics(name, job_type, metric_name):
  metrics_cmd = "arena logs --tail=50 %s | grep -e '%s=' -e '%s:' | tail -1" % (name, metric_name, metric_name)
  metric = 0
  logging.info("search metric_name %s" % (metric_name))
  try:
    import re
    output = subprocess.check_output(metrics_cmd, stderr=subprocess.STDOUT, shell=True)
    result = output.decode().strip()
    split_unit=''
    if metric_name+"=" in result:
        split_unit="="
    elif metric_name+":" in result:
        split_unit=":"
    else:
        return 0
    array = result.split("%s%s" % (metric_name, split_unit))
    if len(array) > 0:
      logging.info(array)
      result = re.findall(r'\d+\.*\d*',array[-1])
      if len(result) > 0:
        metric = float(result[0])
      else:
        logging.warning("Failed to parse metric from %s" % (array[-1]))
        metric = 0
  except Exception as e:
    logging.warning("Failed to get job status due to" + e)
    return 0

  return metric

def get_job_status(name, job_type):
  get_cmd = "arena get %s --type %s | grep -i STATUS:|awk -F: '{print $NF}'" % (name, job_type)
  status = ""
  try:
    output=subprocess.check_output(get_cmd, stderr=subprocess.STDOUT, shell=True)
    status = output.decode()
    status = status.strip()
  except subprocess.CalledProcessError as e:
    logging.warning("Failed to get job status due to" + e)

  return status

def _get_tensorboard_url(name, job_type):
  get_cmd = "arena get %s --type %s | tail -1" % (name, job_type)
  url = "N/A"
  try:
    output = subprocess.check_output(get_cmd, stderr=subprocess.STDOUT, shell=True)
    url = output.decode()
  except subprocess.CalledProcessError as e:
    logging.warning("Failed to get job status due to" + e)

  return url