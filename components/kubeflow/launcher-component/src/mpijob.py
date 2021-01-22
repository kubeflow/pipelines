# -*- coding: utf-8 -*-
"""
@author: chongchuanbing
"""

import logging
import json
import launcher_crd

from common import Status


class MpiJobComponent(launcher_crd.K8sCR):
    def __init__(self, args):
        self.launcher_failed_count = 0
        self.worker_failed_count = 0
        self.launcher_succeeded_count = 0
        self.worker_succeeded_count = 0
        self.launcher_pod_name_dict = {}
        self.worker_pod_name_dict = {}
        self.expected_conditions = ["Succeeded", "Failed"]
        self.launcher_replicas = args.spec['spec']['mpiReplicaSpecs']['Launcher']['replicas']
        self.worker_replicas = args.spec['spec']['mpiReplicaSpecs']['Worker']['replicas']

        logging.info('MPIJob. launcher replicas: %d, worker replicas: %d' % (self.launcher_replicas,
                                                                             self.worker_replicas))

        super(MpiJobComponent, self).__init__(args)

    def enable_watch(self):
        return True

    def get_watch_resources(self):
        label_selector = 'mpi-job-name={}'.format(self.crd_component_name)
        params = {
            'namespace': self.crd_component_namespace,
            'label_selector': label_selector,
            # 'watch': True
        }
        return self.core.list_namespaced_pod, params

    def watch_resources_callback(self, v1_pod):
        phase = v1_pod.status.phase
        replica_type = v1_pod.metadata.labels.get('mpi-job-role', '')
        pod_namespace = v1_pod.metadata.namespace
        pod_name = v1_pod.metadata.name

        if 'launcher' == replica_type:
            self.launcher_pod_name_dict[pod_name] = pod_namespace
        elif 'worker' == replica_type:
            self.worker_pod_name_dict[pod_name] = pod_namespace

        if 'Failed' == phase:
            reason = v1_pod.status.reason

            pod_logs = self.core.read_namespaced_pod_log(name=pod_name,
                                                         namespace=pod_namespace,
                                                         pretty="True",
                                                         tail_lines=self.args.tail_lines)
            logging.warning('Pod %s.%s Failed, replica_type: %s, Reason: %s. Log: \n%s' % (pod_namespace,
                                                                                           pod_name, replica_type,
                                                                                           reason,
                                                                                           pod_logs))

            if 'launcher' == replica_type:
                self.launcher_failed_count += 1
            elif 'worker' == replica_type:
                self.worker_failed_count += 1

            logging.warning(
                'Launcher All:%d, Succeed: %d, Failed: %d; '
                'Worker All: %d, Succeed: %d, Failed: %d' % (self.launcher_replicas,
                                                             self.launcher_succeeded_count,
                                                             self.launcher_failed_count,
                                                             self.worker_replicas,
                                                             self.worker_succeeded_count,
                                                             self.worker_failed_count))
        elif 'Succeeded' == phase:
            if 'launcher' == replica_type:
                self.launcher_succeeded_count += 1
            elif 'worker' == replica_type:
                self.worker_succeeded_count += 1

            logging.warning(
                'Launcher All:%d, Succeed: %d, Failed: %d; '
                'Worker All: %d, Succeed: %d, Failed: %d' % (self.launcher_replicas,
                                                             self.launcher_succeeded_count,
                                                             self.launcher_failed_count,
                                                             self.worker_replicas,
                                                             self.worker_succeeded_count,
                                                             self.worker_failed_count))

        return self.__judge_status()

    def conditions_judge(self, inst):
        return Status.Running, ''

    def __judge_status(self):
        if self.launcher_replicas == self.launcher_failed_count:
            self.__delete_worker()
            return Status.Failed, 'Launcher all failed'

        if self.worker_failed_count > 0 and \
                self.worker_succeeded_count + self.worker_failed_count == self.worker_replicas:
            self.__delete_launcher()
            return Status.Succeed, 'partial failure'
        elif self.launcher_succeeded_count == self.launcher_replicas:
            return Status.Succeed, 'Launcher all succeed'
        return Status.Running, ''

    def __delete_launcher(self):
        for (name, namespace) in self.launcher_pod_name_dict.items():
            self.delete_pod(name, namespace)

    def __delete_worker(self):
        for (name, namespace) in self.worker_pod_name_dict.items():
            self.delete_pod(name, namespace)

    def __expected_conditions_deal(self, current_inst):

        pass