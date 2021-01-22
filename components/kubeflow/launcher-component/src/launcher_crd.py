# -*- coding: utf-8 -*-
"""
@author: chongchuanbing
"""

import datetime
import json
import logging
import time
import os
import asyncio
import threading

from kubernetes import client as k8s_client
from kubernetes.client import rest
from kubernetes import config
from kubernetes import watch

from common import Status

logging.basicConfig(
    format='%(asctime)s %(filename)s[line:%(lineno)d] %(levelname)s %(message)s'
)


class K8sCR(object):
    def __init__(self, args):
        config.load_incluster_config()
        self.ioloop = asyncio.new_event_loop()
        self.__core_w = watch.Watch()
        self.__crd_w = watch.Watch()

        self.__run_status = Status.Succeed

        self.args = args
        self.group = args.group
        self.plural = args.plural
        self.version = args.version
        self.__api_client = k8s_client.ApiClient()
        self.custom = k8s_client.CustomObjectsApi(self.__api_client)
        self.core = k8s_client.CoreV1Api(self.__api_client)

        self.current_pod_name = os.environ.get('POD_NAME') or os.environ.get("HOSTNAME")
        self.current_pod_namespace = os.environ.get('POD_NAMESPACE')

        self.__init_current_pod()

        self.crd_component_namespace = args.spec.get('metadata', {}).get('namespace', None)
        self.crd_component_name = None
        if not self.crd_component_namespace and self.current_pod_namespace:
            args.spec['metadata']['namespace'] = self.current_pod_namespace
            self.crd_component_namespace = self.current_pod_namespace

    def __init_current_pod(self):
        '''
        Init current pod info by api
        :return:
        '''
        if self.current_pod_name:
            try:
                current_pod = self.core.read_namespaced_pod(name=self.current_pod_name,
                                                namespace=self.current_pod_namespace,
                                                pretty='true')
            except Exception as e:
                logging.error("There was a problem getting info for %s/%s %s in namespace %s; Exception: %s",
                              self.current_pod_name, self.current_pod_namespace, e)
                self.current_pod_name = None

            if current_pod:
                self.current_pod_uid = current_pod.metadata.uid
                self.current_pod_api_version = current_pod.api_version
                self.current_pod_kind = current_pod.kind

                logging.info(
                    'current pod name: %s, namespace: %s, apiVersion: %s, kind: %s' % (
                    self.current_pod_name, self.current_pod_namespace,
                    self.current_pod_api_version, self.current_pod_kind))

    def conditions_judge(self, cr_object):
        '''
        :param cr_object:
        :return: 0: Running; -1: Failed; 1: Succeed
        '''
        pass

    def __expected_conditions_deal(self, current_inst):
        pass

    def enable_watch(self):
        return False

    def get_watch_resources(self):
        '''
        Get watched resources you want.
        :return:
        '''
        return None

    def watch_resources_callback(self, v1_pod):
        '''
        :param v1_pod: V1Pod
        :return:
        '''
        pass

    def __set_owner_reference(self, spec):
        '''
        Set ownerReference to crd.
        :param spec:
        :return:
        '''
        if self.current_pod_name:
            owner_reference = {
                "apiVersion": self.current_pod_api_version,
                "blockOwnerDeletion": True,
                "controller": True,
                "kind": self.current_pod_kind,
                "name": self.current_pod_name,
                "uid": self.current_pod_uid
            }
            spec['metadata']['ownerReferences'] = [
                owner_reference
            ]
        return spec

    async def __watch(self):
        logging.info("__watch")
        f, params = self.get_watch_resources()

        for event in self.__core_w.stream(f, **params):
            logging.info(
                "Watch Event: %s %s %s.%s phase: %s" % (
                    event['type'],
                    event['object'].kind,
                    event['object'].metadata.namespace,
                    event['object'].metadata.name,
                    event['object'].status.phase))

            try:
                condition_status, condition = self.watch_resources_callback(event['object'])
            except Exception as e:
                logging.info("%s %s.%s callback Exception. Msg: \n%s.",
                             event['object'].kind, event['object'].metadata.namespace, event['object'].metadata.name, e)

            if Status.Succeed == condition_status:
                logging.info("%s/%s %s.%s Succeed. Msg: %s.",
                             self.group, self.plural, self.crd_component_namespace, self.crd_component_name, condition)
                self.__expected_conditions_deal(event['object'])

                if self.args.delete_after_done:
                    self.__delete_crd(self.crd_component_name, self.crd_component_namespace)

                self.__core_w.stop()
                self.__crd_w.stop()
                logging.info('watch stopped')
                # self.ioloop.stop()
            elif Status.Failed == condition_status:
                logging.error("%s/%s %s.%s Failed. Msg: %s",
                              self.group, self.plural, self.crd_component_namespace, self.crd_component_name, condition)
                if self.args.exception_clear:
                    self.__delete_crd(self.crd_component_name, self.crd_component_namespace)

                self.__core_w.stop()
                self.__crd_w.stop()
                self.__run_status = Status.Failed
                logging.info('watch stopped')
            # await asyncio.sleep(0)
        logging.info("__watch end")

    async def __watch_crd(self):
        '''
        Watch CRD resource. For example, TFJob PytorchJob Experiment
        :return:
        '''
        logging.info("__watch_crd")
        field_selector = 'metadata.name={}'.format(self.crd_component_name)
        for event in self.__crd_w.stream(self.custom.list_namespaced_custom_object,
                                   group=self.group,
                                   version=self.version,
                                   namespace=self.crd_component_namespace,
                                   plural=self.plural,
                                   field_selector=field_selector,
                                   # watch=True
                                   ):
            logging.info(
                "Watch CRD Event: %s %s %s.%s" % (
                    event['type'],
                    event['object'].get('kind', ''),
                    event['object'].get('metadata', {}).get('namespace', ''),
                    event['object'].get('metadata', {}).get('name', '')))

            if event["type"] == "DELETED":
                logging.warning("Crd has been deleted. exit...")
                self.__core_w.stop()
                self.__crd_w.stop()
                # self.ioloop.stop()
                logging.info('watch stopped')
                self.__run_status = Status.Failed

            try:
                condition_status, condition = self.conditions_judge(event['object'])
            except Exception as e:
                logging.error("%s/%s %s.%s Exception. Msg: \n%s",
                              self.group, self.plural, self.crd_component_namespace, self.crd_component_name, e)
                self.__delete_crd(self.crd_component_name, self.crd_component_namespace)

                self.__core_w.stop()
                self.__crd_w.stop()
                # self.ioloop.stop()
                logging.info('watch stopped')
                self.__run_status = Status.Failed

            if Status.Succeed == condition_status:
                logging.info("%s/%s %s.%s Succeed. Msg: %s.",
                             self.group, self.plural, self.crd_component_namespace, self.crd_component_name, condition)
                self.__expected_conditions_deal(event['object'])

                if self.args.delete_after_done:
                    self.__delete_crd(self.crd_component_name, self.crd_component_namespace)

                self.__core_w.stop()
                self.__crd_w.stop()
                logging.info('watch stopped')
                # self.ioloop.stop()
            elif Status.Failed == condition_status:
                logging.error("%s/%s %s.%s Failed. Msg: %s",
                              self.group, self.plural, self.crd_component_namespace, self.crd_component_name, condition)
                if self.args.exception_clear:
                    self.__delete_crd(self.crd_component_name, self.crd_component_namespace)

                self.__core_w.stop()
                self.__crd_w.stop()
                logging.info('watch stopped')
                # self.ioloop.stop()
                self.__run_status = Status.Failed
            else:
                logging.info("%s/%s %s.%s Running. Msg: %s.",
                             self.group, self.plural, self.crd_component_namespace, self.crd_component_name,
                             condition)

            # await asyncio.sleep(0)
        logging.info("__watch_crd end")

    def start_loop(self, loop):
        asyncio.set_event_loop(loop)
        task = loop.create_task(self.__watch())
        try:
            # loop.run_forever()
            loop.run_until_complete(task)
        finally:
            loop.stop()
            loop.close()
            logging.info('loop stopped')

    def deal(self):
        spec = self.__set_owner_reference(self.args.spec)

        create_response = self.__create(spec)

        logging.info(create_response)

        self.crd_component_name = create_response['metadata']['name']
        logging.info('crd component instance name: %s, namespace: %s' % (self.crd_component_name,
                                                                         self.crd_component_namespace))

        ioloop_thread = asyncio.new_event_loop()
        t = threading.Thread(target=self.start_loop, args=(ioloop_thread,))
        t.start()

        task = self.ioloop.create_task(self.__watch_crd())
        try:
            # self.ioloop.run_forever()
            self.ioloop.run_until_complete(task)
            logging.info('run_until_complete end')
        finally:
            logging.info('ioloop stopped')
            self.ioloop.stop()
            self.ioloop.close()

        # ioloop_thread.stop()
        logging.info('end')
        exit(0) if Status.Succeed == self.__run_status else exit(1)

        # if self.enable_watch():
        #     self.ioloop.create_task(self.__watch())
        # self.ioloop.create_task(self.__watch_crd())
        # try:
        #     self.ioloop.run_forever()
        # finally:
        #     self.ioloop.close()

    def __create(self, spec):
        '''
        Create a CR.
        :param spec: The spec for the CR.
        :return:
        '''
        name = spec['metadata']['name'] if 'name' in spec['metadata'] else spec['metadata']['generateName']
        try:
            # Create a Resource
            namespace = spec["metadata"].get("namespace", "default")

            logging.info("Creating %s/%s %s in namespace %s.",
                         self.group, self.plural, name, namespace)

            api_response = self.custom.create_namespaced_custom_object(
              self.group, self.version, namespace, self.plural, spec, pretty='True')

            logging.info("Created %s/%s %s in namespace %s.",
                         self.group, self.plural, api_response['metadata']['name'], namespace)

            return api_response
        except rest.ApiException as e:
            self.__log_and_raise_exception(e, "create")

    def __delete_crd(self, name, namespace):
        try:
            body = {
                # Set garbage collection so that CR won't be deleted until all
                # owned references are deleted.
                "propagationPolicy": "Foreground",
            }
            logging.info("Deleteing %s/%s %s in namespace %s.",
                         self.group, self.plural, name, namespace)

            api_response = self.custom.delete_namespaced_custom_object(
                self.group,
                self.version,
                namespace,
                self.plural,
                name,
                body)

            logging.info("Deleted %s/%s %s in namespace %s.",
                         self.group, self.plural, name, namespace)
            return api_response
        except rest.ApiException as e:
            self.__log_and_raise_exception(e, "delete")

    def delete_pod(self, name, namespace):
        try:
            self.core.delete_namespaced_pod(name, namespace, async_req=True)
            logging.warning('delete pod, name: %s.%s' % (namespace, name))
        except Exception as e:
            logging.error('delete pod %s.%s Exception. Msg: \n%s' % (namespace, name, e))

    def __log_and_raise_exception(self, ex, action):
        message = ""
        if ex.message:
            message = ex.message
        if ex.body:
            try:
                body = json.loads(ex.body)
                message = body.get("message")
            except ValueError:
                logging.error("Exception when %s %s/%s: %s", action, self.group, self.plural, ex.body)
                raise

        logging.error("Exception when %s %s/%s: %s", action, self.group, self.plural, ex.body)
        raise ex

