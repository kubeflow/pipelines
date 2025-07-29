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

    if cpu != '0':
        options.extend(['--cpu', str(cpu)])

    if memory != '0':
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
        if ":" in d:
            options.append("--data={0}".format(d))
        else:
            logging.info("--data={0} is illegal, skip.".format(d))

    if len(env) > 0:
      for e in env:
        if "=" in e:
            options.append("--env={0}".format(e))
        else:
            logging.info("--env={0} is illegal, skip.".format(e))


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
    rdma = args.rdma

    commandArray = [
    'arena', 'submit', 'mpijob',
    '--name={0}'.format(name),
    '--workers={0}'.format(workers),
    '--image={0}'.format(image),
    ]

    if rdma.lower() == "true":
       commandArray.append("--rdma")

    commandArray.extend(generate_options(args))

    return commandArray, "mpijob"
