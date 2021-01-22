# -*- coding: utf-8 -*-
"""
@author: chongchuanbing
"""

import argparse
from distutils.util import strtobool
import logging
import yaml
from experiment import ExperimentComponent
from tfjob import TFJobComponent
from pytorchjob import PytorchJobComponent
from mpijob import MpiJobComponent


def yaml_load(str):
    if '' == str or str is None:
        return None
    try:
        return yaml.safe_load(str)
    except:
        return None


class StandardFactory:
    @staticmethod
    def get_factory(spec_type, args):
        if spec_type == 'TFJob':
            return TFJobComponent(args)
        elif spec_type == 'PyTorchJob':
            return PytorchJobComponent(args)
        elif spec_type == 'Experiment':
            return ExperimentComponent(args)
        elif spec_type == 'MPIJob':
            return MpiJobComponent(args)
        raise TypeError('Unknown Factory.')


if __name__== "__main__":
    parser = argparse.ArgumentParser(description='Kubeflow CRD Component launcher')
    parser.add_argument('--group', type=str,
                      help='crd group.')
    parser.add_argument('--plural', type=str,
                      help='crd plural.')
    parser.add_argument('--version', type=str,
                      default='v1',
                      help='crd version.')
    parser.add_argument('--spec', type=yaml_load,
                      default={},
                      help='crd spec.')
    parser.add_argument('--delete_after_done', type=strtobool,
                      default=False,
                      help='When crd done, delete the crd automatically if it is True.')
    parser.add_argument('--exception_clear', type=strtobool,
                        default=True,
                        help='When crd run failed, delete the crd automatically if it is True.')
    parser.add_argument('--tail_lines', type=int,
                        default=30,
                        help='When pod failed, the number of printted logs')

    parser.add_argument('--output_file', type=str,
                        default='/output.txt',
                        help='The file which stores the best trial of the experiment.')

    args = parser.parse_args()

    logging.getLogger().setLevel(logging.INFO)

    logging.info('Generating crd template.')

    spec_type = args.spec['kind']
    crd_component = StandardFactory.get_factory(spec_type, args)

    crd_component.deal()
