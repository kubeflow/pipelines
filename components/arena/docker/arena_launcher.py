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

def setup_custom_logging():
  logging.basicConfig(format='%(asctime)s %(levelname)-8s %(message)s', 
                      level=logging.INFO,
                      datefmt='%Y-%m-%d %H:%M:%S')

def _submit_job(command):
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

def _wait_job_done(name, job_type, timeout):
  end_time = datetime.datetime.now() + timeout
  logging.info("expect done time: {0}".format(end_time))
  status = _get_job_status(name, job_type)
  while _is_active_status(status):
    if datetime.datetime.now() > end_time:
      timeoutMsg = "Timeout waiting for job {0} with job type {1} completing.".format(name ,job_type)
      logging.error(timeoutMsg)
      raise Exception(timeoutMsg)
    time.sleep(3)
    status = _get_job_status(name, job_type)
  logging.info("job {0} with type {1} status is {2}".format(name, job_type, status))

def _wait_job_running(name, job_type, timeout):
  end_time = datetime.datetime.now() + timeout
  logging.info("expect running time: {0}".format(end_time))
  status = _get_job_status(name, job_type)
  while _is_pending_status(status):
    if datetime.datetime.now() > end_time:
      timeoutMsg = "Timeout waiting for job {0} with job type {1} running.".format(name ,job_type)
      logging.error(timeoutMsg)
      raise Exception(timeoutMsg)
    time.sleep(3)
    status = _get_job_status(name, job_type)
  logging.info("job {0} with type {1} status is {2}".format(name, job_type, status))

def _job_logging(name, job_type):
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

def _collect_metrics(name, job_type, metric_name):
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
      metric = float(array[-1])
  except Exception as e:
    logging.warning("Failed to get job status due to" + e)
    return 0

  return metric

def _get_job_status(name, job_type):
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

# Generate common options
def generate_options(args):
    gpus = args.gpus
    cpu = args.cpu
    memory = args.memory
    tensorboard = args.tensorboard
    output_data = args.output_data
    data = args.data
    env = args.env
    tensorboard_image = args.tensorboard_image
    tensorboard = str2bool(args.tensorboard)
    log_dir = args.log_dir
    sync_source = args.sync_source

    options = []

    if gpus > 0:
        options.extend(['--gpus', str(gpus)])

    if cpu > 0:
        options.extend(['--cpu', str(cpu)])

    if memory >0:
        options.extend(['--memory', str(memory)])

    if tensorboard_image != "tensorflow/tensorflow:1.12.0":
        options.extend(['--tensorboardImage', tensorboard_image])    

    if tensorboard:
        options.append("--tensorboard")

    if os.path.isdir(args.log_dir):  
        options.extend(['--logdir', args.log_dir])
    else:
        logging.info("skip log dir :{0}".format(args.log_dir))

    if len(data) > 0:
      for d in data:
        options.append("--data={0}".format(d))

    if len(env) > 0:
      for e in env:
        options.append("--env={0}".format(e))

    if len(args.workflow_name) > 0:
        options.append("--env=WORKFLOW_NAME={0}".format(args.workflow_name))

    if len(args.step_name) > 0:
        options.append("--env=STEP_NAME={0}".format(args.step_name))

    if len(sync_source) > 0:
      if not sync_source.endswith(".git"):
        raise ValueError("sync_source must be an http git url")
      options.extend(['--sync-mode','git'])
      options.extend(['--sync-source',sync_source])

    return options



# Generate standalone job
def generate_job_command(args):
    name = args.name
    image = args.image

    commandArray = [
    'arena', 'submit', 'tfjob',
    '--name={0}'.format(name),
    '--image={0}'.format(image),
    ]

    commandArray.extend(generate_options(args))

    return commandArray, "tfjob"

# Generate mpi job
def generate_mpjob_command(args):
    name = args.name
    workers = args.workers
    image = args.image

    commandArray = [
    'arena', 'submit', 'mpijob',
    '--name={0}'.format(name),
    '--workers={0}'.format(workers),
    '--image={0}'.format(image),
    ]

    if rdma:
       commandArray.append("--rdma")

    commandArray.extend(generate_options(args))

    return commandArray, "mpijob"

def str2bool(v):
  return v.lower() in ("yes", "true", "t", "1")


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
                      help='Time in hours to wait for the Job submitted by arena to complete')
  # parser.add_argument('--command', type=str)
  parser.add_argument('--output-dir', type=str, default='')
  parser.add_argument('--output-data', type=str, default='None')
  parser.add_argument('--log-dir', type=str, default='')
  
  parser.add_argument('--image', type=str)
  parser.add_argument('--gpus', type=int, default=0)
  parser.add_argument('--cpu', type=int, default=0)
  parser.add_argument('--memory', type=int, default=0)
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
  
  _submit_job(command)
  
  #with open('/mlpipeline-ui-metadata.json', 'w') as f:
  #    json.dump(metadata, f)

  
  succ = True

  # wait for job done
  # _wait_job_done(fullname, job_type, datetime.timedelta(minutes=timeout_hours))
  _wait_job_running(fullname, job_type, datetime.timedelta(minutes=30))

  rc = _job_logging(fullname, job_type)
  logging.info("rc: {0}".format(rc))
  
  _wait_job_done(fullname, job_type, datetime.timedelta(hours=timeout_hours))
  
  status = _get_job_status(fullname, job_type)

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
        value = _collect_metrics(fullname, job_type, metric_name)
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
