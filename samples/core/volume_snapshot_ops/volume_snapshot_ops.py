import kfp.dsl as dsl

from kubernetes import client

client.CoreV1Api


@dsl.pipeline(name='volume-example')
def volume_pipeline(url: str):
    # create PersistentVolumeClaim (PVC)
    create_volume_task = dsl.VolumeOp(
        name='{{{{workflow.name}}}}-kfp-pvc',
        resource_name='vol1',
        volume_name='task-pv-volume',
        storage_class='manual',
        size='1Gi',
        modes=dsl.VOLUME_MODE_RWO).add_pod_annotation(
            name="pipelines.kubeflow.org/max_cache_staleness", value="P0D")

    # use PVC; write file1.gz to volume
    task_a = dsl.ContainerOp(
        name='step1_ingest',
        image='alpine',
        command=['sh', '-c'],
        arguments=[
            'mkdir /data/step1 && '
            'gsutil cat %s | gzip -c >/data/step1/file1.gz' % url
        ],
        pvolumes={'/data': create_volume_task.volume})

    # # take a snapshot of the PVC with file1.gz
    # snapshot_volume_task = dsl.VolumeSnapshotOp(name='step1_snap',
    #                                             resource_name='step1_snap',
    #                                             volume=task_a.pvolume)

    # # use the same PVC again; unzip file1.gz
    task_b = dsl.ContainerOp(
        name='step2_gunzip',
        image='alpine',
        command=['sh', '-c'],
        arguments=[
            'mkdir /data/step2 && '
            'gunzip /data/step1/file1.gz -c >/data/step2/file1'
        ],
        pvolumes={'/data': create_volume_task.volume},
    )

    # create_volume_task = dsl.VolumeOp(
    #     name='create_volume_from_snapshot',
    #     resource_name='vol1',
    #     size='1Gi',
    #     modes=dsl.VOLUME_MODE_RWO,
    #     data_source=snapshot_volume_task.snapshot)


if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from kfp import compiler

    from kfp import Client

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.yaml')
    print(ir_file)
    compiler.Compiler().compile(pipeline_func=volume_pipeline,
                                package_path=ir_file)
    # pipeline_name = __file__.split('/')[-1].replace('_',
    #                                                 '-').replace('.py', '')
    # display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
    # job_id = f'{pipeline_name}-{display_name}'
    # endpoint = 'https://75167a6cffcb723c-dot-us-central1.pipelines.googleusercontent.com'
    # kfp_client = Client(host=endpoint)
    # run = kfp_client.create_run_from_pipeline_package(ir_file, arguments={})
    # url = f'{endpoint}/#/runs/details/{run.run_id}'
    # print(url)
    # webbrowser.open_new_tab(url)
