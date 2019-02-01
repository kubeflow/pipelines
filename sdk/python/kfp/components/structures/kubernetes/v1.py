__all__ = [
    'Container',
    'PodArgoSubset',
]


from collections import OrderedDict

from typing import Any, Dict, List, Mapping, Optional, Sequence, Union

from ...modelbase import ModelBase


class EnvVar(ModelBase):
    _serialized_names = {
        'value_from': 'valueFrom',
    }
    def __init__(self,
        name: str,
        value: Optional[str] = None,
        #value_from: Optional[EnvVarSource] = None, #TODO: Add if needed
    ):
        super().__init__(locals())


class ExecAction(ModelBase):
    def __init__(self,
        command: List[str],
    ):
        super().__init__(locals())


class Handler(ModelBase):
    _serialized_names = {
        'http_get': 'httpGet',
        'tcp_socket': 'tcpSocket',
    }
    def __init__(self,
        exec: Optional[ExecAction] = None,
        #http_get: Optional[HTTPGetAction] = None, #TODO: Add if needed
        #tcp_socket: Optional[TCPSocketAction] = None, #TODO: Add if needed
    ):
        super().__init__(locals())


class Lifecycle(ModelBase):
    _serialized_names = {
        'post_start': 'postStart',
        'pre_stop': 'preStop',
    }
    def __init__(self,
        post_start: Optional[Handler] = None,
        pre_stop: Optional[Handler] = None,
    ):
        super().__init__(locals())


class VolumeMount(ModelBase):
    _serialized_names = {
        'mount_path': 'mountPath',
        'mount_propagation': 'mountPropagation',
        'read_only': 'readOnly',
        'sub_path': 'subPath',
    }
    def __init__(self,
        name: str,
        mount_path: str,
        mount_propagation: Optional[str] = None,
        read_only: Optional[bool] = None,
        sub_path: Optional[str] = None,
    ):
        super().__init__(locals())


class ResourceRequirements(ModelBase):
    def __init__(self,
        limits: Optional[Dict[str, str]] = None,
        requests: Optional[Dict[str, str]] = None,
    ):
        super().__init__(locals())


class ContainerPort(ModelBase):
    _serialized_names = {
        'container_port': 'containerPort',
        'host_ip': 'hostIP',
        'host_port': 'hostPort',
    }
    def __init__(self,
        container_port: int,
        host_ip: Optional[str] = None,
        host_port: Optional[int] = None,
        name: Optional[str] = None,
        protocol: Optional[str] = None,
    ):
        super().__init__(locals())


class VolumeDevice(ModelBase):
    _serialized_names = {
        'device_path': 'devicePath',
    }
    def __init__(self,
        device_path: str,
        name: str,
    ):
        super().__init__(locals())


class Probe(ModelBase):
    _serialized_names = {
        'failure_threshold': 'failureThreshold',
        'http_get': 'httpGet',
        'initial_delay_seconds': 'initialDelaySeconds',
        'period_seconds': 'periodSeconds',
        'success_threshold': 'successThreshold',
        'tcp_socket': 'tcpSocket',
        'timeout_seconds': 'timeoutSeconds'
    }
    def __init__(self,
        exec: Optional[ExecAction] = None,
        failure_threshold: Optional[int] = None,
        #http_get: Optional[HTTPGetAction] = None, #TODO: Add if needed
        initial_delay_seconds: Optional[int] = None,
        period_seconds: Optional[int] = None,
        success_threshold: Optional[int] = None,
        #tcp_socket: Optional[TCPSocketAction] = None, #TODO: Add if needed
        timeout_seconds: Optional[int] = None,
    ):
        super().__init__(locals())


class SecurityContext(ModelBase):
    _serialized_names = {
        'allow_privilege_escalation': 'allowPrivilegeEscalation',
        'capabilities': 'capabilities',
        'privileged': 'privileged',
        'read_only_root_filesystem': 'readOnlyRootFilesystem',
        'run_as_group': 'runAsGroup',
        'run_as_non_root': 'runAsNonRoot',
        'run_as_user': 'runAsUser',
        'se_linux_options': 'seLinuxOptions'
    }
    def __init__(self,
        allow_privilege_escalation: Optional[bool] = None,
        #capabilities: Optional[Capabilities] = None, #TODO: Add if needed
        privileged: Optional[bool] = None,
        read_only_root_filesystem: Optional[bool] = None,
        run_as_group: Optional[int] = None,
        run_as_non_root: Optional[bool] = None,
        run_as_user: Optional[int] = None,
        #se_linux_options: Optional[SELinuxOptions] = None, #TODO: Add if needed
    ):
        super().__init__(locals())


class Container(ModelBase):
    _serialized_names = {
        'env_from': 'envFrom',
        'image_pull_policy': 'imagePullPolicy',
        'liveness_probe': 'livenessProbe',
        'readiness_probe': 'readinessProbe',
        'security_context': 'securityContext',
        'stdin_once': 'stdinOnce',
        'termination_message_path': 'terminationMessagePath',
        'termination_message_policy': 'terminationMessagePolicy',
        'volume_devices': 'volumeDevices',
        'volume_mounts': 'volumeMounts',
        'working_dir': 'workingDir',
    }

    def __init__(self,
        #Better to set at Component level
        image: Optional[str] = None,
        command: Optional[List[str]] = None,
        args: Optional[List[str]] = None,
        env: Optional[List[EnvVar]] = None,

        working_dir: Optional[str] = None, #Not really needed: container should have proper working dir set up

        lifecycle: Optional[Lifecycle] = None, #Can be used to specify pre-exit commands to run TODO: Probably support at Component level.

        #Better to set at Task level
        volume_mounts: Optional[List[VolumeMount]] = None,
        resources: Optional[ResourceRequirements] = None,

        #Might not be used a lot
        ports: Optional[List[ContainerPort]] = None,
        #env_from: Optional[List[EnvFromSource]] = None, #TODO: Add if needed
        volume_devices: Optional[List[VolumeDevice]] = None,

        #Probably not needed
        name: Optional[str] = None, #Required by k8s schema, but not Argo.
        image_pull_policy: Optional[str] = None,
        liveness_probe: Optional[Probe] = None,
        readiness_probe: Optional[Probe] = None,
        security_context: Optional[SecurityContext] = None,
        stdin: Optional[bool] = None,
        stdin_once: Optional[bool] = None,
        termination_message_path: Optional[str] = None,
        termination_message_policy: Optional[str] = None,
        tty: Optional[bool] = None,
    ):
        super().__init__(locals())


#class NodeAffinity(ModelBase):
#    _serialized_names = {
#        'preferred_during_scheduling_ignored_during_execution': 'preferredDuringSchedulingIgnoredDuringExecution',
#        'required_during_scheduling_ignored_during_execution': 'requiredDuringSchedulingIgnoredDuringExecution',
#    }
#    def __init__(self,
#        preferred_during_scheduling_ignored_during_execution: Optional[List[PreferredSchedulingTerm]] = None,
#        required_during_scheduling_ignored_during_execution: Optional[NodeSelector] = None,
#    ):
#        super().__init__(locals())


#class Affinity(ModelBase):
#    _serialized_names = {
#        'node_affinity': 'nodeAffinity',
#        'pod_affinity': 'podAffinity',
#        'pod_anti_affinity': 'podAntiAffinity',
#    }
#    def __init__(self,
#        node_affinity: Optional[NodeAffinity] = None,
#        #pod_affinity: Optional[PodAffinity] = None, #TODO: Add if needed
#        #pod_anti_affinity: Optional[PodAntiAffinity] = None, #TODO: Add if needed
#    ):
#        super().__init__(locals())


class Toleration(ModelBase):
    _serialized_names = {
        'toleration_seconds': 'tolerationSeconds',
    }
    def __init__(self,
        effect: Optional[str] = None,
        key: Optional[str] = None,
        operator: Optional[str] = None,
        toleration_seconds: Optional[int] = None,
        value: Optional[str] = None,
    ):
        super().__init__(locals())


class KeyToPath(ModelBase):
    def __init__(self,
        key: str,
        path: str,
        mode: Optional[int] = None,
    ):
        super().__init__(locals())


class SecretVolumeSource(ModelBase):
    _serialized_names = {
        'default_mode': 'defaultMode',
        'secret_name': 'secretName'
    }
    def __init__(self,
        default_mode: Optional[int] = None,
        items: Optional[List[KeyToPath]] = None,
        optional: Optional[bool] = None,
        secret_name: Optional[str] = None,
    ):
        super().__init__(locals())


class NFSVolumeSource(ModelBase):
    _serialized_names = {
        'read_only': 'readOnly',
    }
    def __init__(self,
        path: str,
        server: str,
        read_only: Optional[bool] = None,
    ):
        super().__init__(locals())


class PersistentVolumeClaimVolumeSource(ModelBase):
    _serialized_names = {
        'claim_name': 'claimName',
        'read_only': 'readOnly'
    }
    def __init__(self,
        claim_name: str,
        read_only: Optional[bool] = None,
    ):
        super().__init__(locals())


class Volume(ModelBase):
    _serialized_names = {
        'aws_elastic_block_store': 'awsElasticBlockStore',
        'azure_disk': 'azureDisk',
        'azure_file': 'azureFile',
        'cephfs': 'cephfs',
        'cinder': 'cinder',
        'config_map': 'configMap',
        'downward_api': 'downwardAPI',
        'empty_dir': 'emptyDir',
        'fc': 'fc',
        'flex_volume': 'flexVolume',
        'flocker': 'flocker',
        'gce_persistent_disk': 'gcePersistentDisk',
        'git_repo': 'gitRepo',
        'glusterfs': 'glusterfs',
        'host_path': 'hostPath',
        'iscsi': 'iscsi',
        'name': 'name',
        'nfs': 'nfs',
        'persistent_volume_claim': 'persistentVolumeClaim',
        'photon_persistent_disk': 'photonPersistentDisk',
        'portworx_volume': 'portworxVolume',
        'projected': 'projected',
        'quobyte': 'quobyte',
        'rbd': 'rbd',
        'scale_io': 'scaleIO',
        'secret': 'secret',
        'storageos': 'storageos',
        'vsphere_volume': 'vsphereVolume'
    }

    def __init__(self,
        name: str,
        secret: Optional[SecretVolumeSource] = None,
        nfs: Optional[NFSVolumeSource] = None,
        persistent_volume_claim: Optional[PersistentVolumeClaimVolumeSource] = None,

        #No validation for these volume types
        aws_elastic_block_store: Optional[Mapping] = None, #AWSElasticBlockStoreVolumeSource,
        azure_disk: Optional[Mapping] = None, #AzureDiskVolumeSource,
        azure_file: Optional[Mapping] = None, #AzureFileVolumeSource,
        cephfs: Optional[Mapping] = None, #CephFSVolumeSource,
        cinder: Optional[Mapping] = None, #CinderVolumeSource,
        config_map: Optional[Mapping] = None, #ConfigMapVolumeSource,
        downward_api: Optional[Mapping] = None, #DownwardAPIVolumeSource,
        empty_dir: Optional[Mapping] = None, #EmptyDirVolumeSource,
        fc: Optional[Mapping] = None, #FCVolumeSource,
        flex_volume: Optional[Mapping] = None, #FlexVolumeSource,
        flocker: Optional[Mapping] = None, #FlockerVolumeSource,
        gce_persistent_disk: Optional[Mapping] = None, #GCEPersistentDiskVolumeSource,
        git_repo: Optional[Mapping] = None, #GitRepoVolumeSource,
        glusterfs: Optional[Mapping] = None, #GlusterfsVolumeSource,
        host_path: Optional[Mapping] = None, #HostPathVolumeSource,
        iscsi: Optional[Mapping] = None, #ISCSIVolumeSource,
        photon_persistent_disk: Optional[Mapping] = None, #PhotonPersistentDiskVolumeSource,
        portworx_volume: Optional[Mapping] = None, #PortworxVolumeSource,
        projected: Optional[Mapping] = None, #ProjectedVolumeSource,
        quobyte: Optional[Mapping] = None, #QuobyteVolumeSource,
        rbd: Optional[Mapping] = None, #RBDVolumeSource,
        scale_io: Optional[Mapping] = None, #ScaleIOVolumeSource,
        storageos: Optional[Mapping] = None, #StorageOSVolumeSource,
        vsphere_volume: Optional[Mapping] = None, #VsphereVirtualDiskVolumeSource,
    ):
        super().__init__(locals())


class PodSpecArgoSubset(ModelBase):
    _serialized_names = {
        'active_deadline_seconds': 'activeDeadlineSeconds',
        'affinity': 'affinity',
        #'automount_service_account_token': 'automountServiceAccountToken',
        #'containers': 'containers',
        #'dns_config': 'dnsConfig',
        #'dns_policy': 'dnsPolicy',
        #'host_aliases': 'hostAliases',
        #'host_ipc': 'hostIPC',
        #'host_network': 'hostNetwork',
        #'host_pid': 'hostPID',
        #'hostname': 'hostname',
        #'image_pull_secrets': 'imagePullSecrets',
        #'init_containers': 'initContainers',
        #'node_name': 'nodeName',
        'node_selector': 'nodeSelector',
        #'priority': 'priority',
        #'priority_class_name': 'priorityClassName',
        #'readiness_gates': 'readinessGates',
        #'restart_policy': 'restartPolicy',
        #'scheduler_name': 'schedulerName',
        #'security_context': 'securityContext',
        #'service_account': 'serviceAccount',
        #'service_account_name': 'serviceAccountName',
        #'share_process_namespace': 'shareProcessNamespace',
        #'subdomain': 'subdomain',
        #'termination_grace_period_seconds': 'terminationGracePeriodSeconds',
        'tolerations': 'tolerations',
        'volumes': 'volumes',
    }
    def __init__(self,
        active_deadline_seconds: Optional[int] = None,
        affinity: Optional[Mapping] = None, #Affinity, #No validation
        #automount_service_account_token: Optional[bool] = None, #Not supported by Argo
        #containers: Optional[List[Container]] = None, #Not supported by Argo
        #dns_config: Optional[PodDNSConfig] = None, #Not supported by Argo
        #dns_policy: Optional[str] = None, #Not supported by Argo
        #host_aliases: Optional[List[HostAlias]] = None, #Not supported by Argo
        #host_ipc: Optional[bool] = None, #Not supported by Argo
        #host_network: Optional[bool] = None, #Not supported by Argo
        #host_pid: Optional[bool] = None, #Not supported by Argo
        #hostname: Optional[str] = None, #Not supported by Argo
        #image_pull_secrets: Optional[List[LocalObjectReference]] = None, #Not supported by Argo
        #init_containers: Optional[List[Container]] = None, #Not supported by Argo
        #node_name: Optional[str] = None, #Not supported by Argo
        node_selector: Optional[Dict[str, str]] = None,
        #priority: Optional[int] = None, #Not supported by Argo
        #priority_class_name: Optional[str] = None, #Not supported by Argo
        #readiness_gates: Optional[List[PodReadinessGate]] = None, #Not supported by Argo
        #restart_policy: Optional[str] = None, #Not supported by Argo
        #scheduler_name: Optional[str] = None, #Not supported by Argo
        #security_context: Optional[PodSecurityContext] = None, #Not supported by Argo
        #service_account: Optional[str] = None, #Not supported by Argo
        #service_account_name: Optional[str] = None, #Not supported by Argo
        #share_process_namespace: Optional[bool] = None, #Not supported by Argo
        #subdomain: Optional[str] = None, #Not supported by Argo
        #termination_grace_period_seconds: Optional[int] = None, #Not supported by Argo
        tolerations: Optional[List[Toleration]] = None,
        volumes: Optional[List[Volume]] = None, #Argo only supports volumes at the Workflow level

        #+Argo features:
        #+Metadata: ArgoMetadata? (Argo version)
        #+RetryStrategy: ArgoRetryStrategy ~= k8s.JobSpec.backoffLimit
        #+Parallelism: int
    ):
        super().__init__(locals())


class ObjectMetaArgoSubset(ModelBase):
    def __init__(self,
        annotations: Optional[Dict[str, str]] = None,
        labels: Optional[Dict[str, str]] = None,
    ):
        super().__init__(locals())


class PodArgoSubset(ModelBase):
    _serialized_names = {
        'api_version': 'apiVersion',
        'kind': 'kind',
        'metadata': 'metadata',
        'spec': 'spec',
        'status': 'status',
    }
    def __init__(self,
        #api_version: Optional[str] = None,
        #kind: Optional[str] = None,
        #metadata: Optional[ObjectMeta] = None,
        metadata: Optional[ObjectMetaArgoSubset] = None,
        #spec: Optional[PodSpec] = None,
        spec: Optional[PodSpecArgoSubset] = None,
        #status: Optional[PodStatus] = None,
    ):
        super().__init__(locals())
