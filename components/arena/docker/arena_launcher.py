"""
Usage:
python arena_launcher.py
    --name=tf-test
    --tensorboard=true
    mpijob
    --gpus=1
    --workers=2
    --image=registry.cn-hangzhou.aliyuncs.com/tensorflow-samples/horovod:0.13.11-tf1.10.0-torch0.4.0-py3.5
    --
    mpirun python /benchmarks/scripts/tf_cnn_benchmarks/tf_cnn_benchmarks.py --model resnet101 --batch_size 64     --variable_update horovod --train_dir=/training_logs --summary_verbosity=3 --save_summaries_steps=10
"""
# TODO: Add unit/integration tests

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
from utils import *
from job_generator import *

def main(argv=None):
  setup_custom_logging()
  import sys
  all_args = sys.argv[1:]
  logging.info("args: {0}".format(' '.join(sys.argv)))
  parser = argparse.ArgumentParser(description='Arena launcher')
  parser.add_argument('--name', type=str,
                      help='The job name to specify.',default=None)
  parser.add_argument('--tensorboard', type=str, default="False")
  parser.add_argument('--rdma', type=str, default="False")
  parser.add_argument('--tensorboard-image', type=str, default='tensorflow/tensorflow:1.12.0')
  parser.add_argument('--timeout-hours', type=int,
                      default=200,
                      help='Time in minutes to wait for the Job submitted by arena to complete')
  parser.add_argument('--pending-timeout-minutes', type=int,
                      default=360,
                      help='Time in hours to wait for the Job submitted by arena from pending to running')
  # parser.add_argument('--command', type=str)
  parser.add_argument('--output-dir', type=str, default='')
  parser.add_argument('--output-data', type=str, default='None')
  parser.add_argument('--log-dir', type=str, default='')
  
  parser.add_argument('--image', type=str)
  parser.add_argument('--gpus', type=int, default=0)
  parser.add_argument('--cpu', type=str, default='0')
  parser.add_argument('--memory', type=str, default='0')
  parser.add_argument('--workers', type=int, default=2)

  parser.add_argument('--env', action='append', type=str, default=[])
  parser.add_argument('--data', action='append', type=str, default=[])
  parser.add_argument('--metric', action='append', type=str, default=[])
  parser.add_argument('--sync-source', type=str, default='')

  parser.add_argument('--workflow-name', type=str, default='')
  parser.add_argument('--step-name', type=str, default='')

  subparsers = parser.add_subparsers(help='arena sub-command help')

  #create the parser for the 'mpijob' command
  parser_mpi = subparsers.add_parser('mpijob', help='mpijob help')
  parser_mpi.set_defaults(func=generate_mpjob_command)

  #create the parser for the 'job' command
  parser_job = subparsers.add_parser('job', help='job help')
  parser_job.set_defaults(func=generate_job_command)


  separator_idx = all_args.index('--')
  launcher_args = all_args[:separator_idx]
  remaining_args = all_args[separator_idx + 1:]
  
  args = parser.parse_args(launcher_args)
  commandArray, job_type = args.func(args)

  args_dict = vars(args)
  if args.name is None:
    logging.error("Please specify the name")
    sys.exit(-1)
  if len(remaining_args) == 0:
    logging.error("Please specify the command.")
    sys.exit(-1)

  internalCommand = ' '.join(remaining_args)

  name = args.name
  fullname = name + datetime.datetime.now().strftime("%Y%M%d%H%M%S")
  timeout_hours = args_dict.pop('timeout_hours')
  logging.info("timeout_hours: {0}".format(timeout_hours))

  enableTensorboard = str2bool(args.tensorboard)

  commandArray.append('"{0}"'.format(internalCommand))
  command = ' '.join(commandArray)
  
  command=command.replace("--name={0}".format(name),"--name={0}".format(fullname))
  
  logging.info('Start training {0}.'.format(command))
  
  submit_job(command)
  
  succ = True

  # wait for job done
  # wait_job_done(fullname, job_type, datetime.timedelta(minutes=timeout_hours))
  pending_timeout_minutes = args.pending_timeout_minutes
  wait_job_running(fullname, job_type, datetime.timedelta(minutes=pending_timeout_minutes))

  rc = job_logging(fullname, job_type)
  logging.info("rc: {0}".format(rc))
  
  wait_job_done(fullname, job_type, datetime.timedelta(hours=timeout_hours))
  
  status = get_job_status(fullname, job_type)

  if status == "SUCCEEDED":
    logging.info("Training Job {0} success.".format(fullname))
    if len(args.metric) > 0:
      metrics_data = {
        'metrics': []
      }
      metric_list = []
      metric_unit="RAW"
      for m in args.metric:
        mArray = m.split(":")
        metric_name = mArray[0]
        if len(mArray) > 1:
          metric_unit = mArray[1]
        logging.info("determine metric name {0} with metric unit {1}".format(metric_name, metric_unit))
        value = collect_metrics(fullname, job_type, metric_name)
        if value > 0:
          import re
          p = re.compile('^[a-z]([-a-z0-9]{0,62}[a-z0-9])?')
          result = p.search(metric_name.lower())
          if result is None:
            logging.info("Failed to parse metric_name {0},skip".format(metric_name))
            continue
          else:
            metric_name=result.group(0)

          metric_data = {
          'name': metric_name.lower(), # The name of the metric. Visualized as the column name in the runs table.
          'numberValue':  value, # The value of the metric. Must be a numeric value.
          'format': metric_unit,   # The optional format of the metric. Supported values are "RAW" (displayed in raw format) and "PERCENTAGE" (displayed in percentage format).
          }
          logging.info("metric data: {0}".format(metric_data))
          metric_list.append(metric_data)
          metrics_data['metrics'] = metric_list
      with open('/mlpipeline-metrics.json', 'w') as f:
        logging.info("metrics: {0}".format(metrics_data))
        json.dump(metrics_data, f)
        logging.info("write down /mlpipeline-metrics.json")
  elif status == "FAILED":
    logging.error("Training Job {0} fail.".format(fullname))
    sys.exit(-1)
  else:
    logging.error("Training Job {0}'s status {1}".format(fullname, status))
    sys.exit(-1)

  # TODO(cheyang): copy the output.txt from training job
  output=""
  with open('/output.txt', 'w') as f:
    f.write(output)

  with open('/workflow-name.txt', 'w') as f:
    f.write(args.workflow_name)

  with open('/step-name.txt', 'w') as f:
    f.write(args.step_name)

  with open('/name.txt', 'w') as f:
    f.write(args.name)

if __name__== "__main__":
  main()
