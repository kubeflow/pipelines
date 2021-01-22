# -*- coding: utf-8 -*-
"""
@author: chongchuanbing
"""

import logging
import launcher_crd

from common import Status


class PytorchJobComponent(launcher_crd.K8sCR):
    def __init__(self, args):
        self.master_failed_count = 0
        self.worker_failed_count = 0
        self.master_succeeded_count = 0
        self.worker_succeeded_count = 0

        self.master_pod_name_dict = {}
        self.worker_pod_name_dict = {}
        self.expected_conditions = ["Succeeded", "Failed"]
        self.master_replicas = args.spec['spec']['pytorchReplicaSpecs']['Master']['replicas']
        self.worker_replicas = args.spec['spec']['pytorchReplicaSpecs']['Worker']['replicas']

        logging.info('PytorchJob. master replicas: %d, worker replicas: %d' % (self.master_replicas, self.worker_replicas))

        super(PytorchJobComponent, self).__init__(args)

    def enable_watch(self):
        return True

    def get_watch_resources(self):
        label_selector = 'pytorch-job-name={}'.format(self.crd_component_name)
        params = {
            'namespace': self.crd_component_namespace,
            'label_selector': label_selector,
            # 'watch': True
        }
        return self.core.list_namespaced_pod, params

    def watch_resources_callback(self, v1_pod):
        phase = v1_pod.status.phase
        pod_namespace = v1_pod.metadata.namespace
        pod_name = v1_pod.metadata.name

        replica_type = v1_pod.metadata.labels.get('pytorch-replica-type', '')

        if 'master' == replica_type:
            self.master_pod_name_dict[pod_name] = pod_namespace
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

            if 'master' == replica_type:
                self.master_failed_count += 1
            elif 'worker' == replica_type:
                self.worker_failed_count += 1

            logging.warning('Master All: %d, Succeed: %d, Failed: %d; '
                            'Worker All: %d, Succeed: %d, Failed: %d' % (self.master_replicas,
                                                                         self.master_succeeded_count,
                                                                         self.master_failed_count,
                                                                         self.worker_replicas,
                                                                         self.worker_succeeded_count,
                                                                         self.worker_failed_count))
        elif 'Succeeded' == phase:
            if 'ps' == replica_type:
                self.master_succeeded_count += 1
            elif 'worker' == replica_type:
                self.worker_succeeded_count += 1

            logging.warning('Master All: %d, Succeed: %d, Failed: %d; '
                            'Worker All: %d, Succeed: %d, Failed: %d' % (self.master_replicas,
                                                                         self.master_succeeded_count,
                                                                         self.master_failed_count,
                                                                         self.worker_replicas,
                                                                         self.worker_succeeded_count,
                                                                         self.worker_failed_count))

        return self.__judge_status()

    def conditions_judge(self, inst):
        return Status.Running, ''

    def __judge_status(self):
        if self.master_replicas == self.master_failed_count:
            self.__delete_worker()
            return Status.Failed, 'Master all failed'
        elif self.worker_replicas == self.worker_failed_count:
            self.__delete_master()
            return Status.Failed, 'Worker all failed'

        if self.worker_failed_count > 0 and \
                self.worker_succeeded_count + self.worker_failed_count == self.worker_replicas:
            self.__delete_master()
            return Status.Succeed, 'partial failure'
        elif self.worker_succeeded_count == self.worker_replicas:
            return Status.Succeed, 'Worker all succeed'
        return Status.Running, ''

    def __delete_master(self):
        for (name, namespace) in self.master_pod_name_dict.items():
            self.delete_pod(name, namespace)

    def __delete_worker(self):
        for (name, namespace) in self.worker_pod_name_dict.items():
            self.delete_pod(name, namespace)

    def __expected_conditions_deal(self, current_inst):

        pass