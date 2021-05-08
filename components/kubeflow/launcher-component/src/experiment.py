# -*- coding: utf-8 -*-
"""
@author: chongchuanbing
"""

import os
import json
import logging
import launcher_crd

from common import Status


class ExperimentComponent(launcher_crd.K8sCR):
    def __init__(self, args):
        self.expected_conditions = ["Succeeded", "Failed"]
        super(ExperimentComponent, self).__init__(args)

    def conditions_judge(self, inst):
        trials_running_count = inst.get('status', {}).get('trialsRunning')
        trials_pending_count = inst.get('status', {}).get('trialsPending')
        if trials_running_count or trials_pending_count:
            return Status.Running, 'Running'

        conditions = inst.get('status', {}).get("conditions")
        if not conditions:
            return Status.Running, "conditions not found"

        if conditions[-1]["type"] in self.expected_conditions:
            return Status.Succeed, conditions[-1]["type"]
        else:
            return Status.Failed, conditions[-1]["type"]

    def __expected_conditions_deal(self, current_inst):
        output_file = self.args.output_file
        param_assignments = current_inst["status"]["currentOptimalTrial"]["parameterAssignments"]
        if not os.path.exists(os.path.dirname(output_file)):
            os.makedirs(os.path.dirname(output_file))
        with open(output_file, 'w') as f:
            f.write(json.dumps(param_assignments))

        logging.info('hp param assignments has been writed to %s' % output_file)
