from typing import List

import kfp
from kfp import dsl


@dsl.component()
def split_ids(ids: str) -> list:
    return ids.split(',')


@dsl.component()
def prepend_id(content: str) -> str:
    print(f"prepending: {content} with 'model_id'")
    return f'model_id_{content}'


@dsl.component()
def consume_ids(ids: List[str]) -> str:
    for id in ids:
        print(f'Consuming: {id}')
    return 'completed'


@dsl.component()
def consume_single_id(id: str) -> str:
    print(f'Consuming single: {id}')
    return 'completed'


@dsl.pipeline()
def collecting_parameters(model_ids: str = '',) -> List[str]:
    ids_split_op = split_ids(ids=model_ids)
    with dsl.ParallelFor(ids_split_op.output) as model_id:
        prepend_id_op = prepend_id(content=model_id)
        prepend_id_op.set_caching_options(False)
        
        consume_single_id_op = consume_single_id(id=prepend_id_op.output)
        consume_single_id_op.set_caching_options(False)
    
    consume_ids_op = consume_ids(ids=dsl.Collected(prepend_id_op.output))
    consume_ids_op.set_caching_options(False)

    return dsl.Collected(prepend_id_op.output)


@dsl.pipeline()
def collected_param_pipeline():
    model_ids = 's1,s2,s3'
    dag = collecting_parameters(model_ids=model_ids)
    dag.set_caching_options(False)
    
    consume_ids_op = consume_ids(ids=dag.output)
    consume_ids_op.set_caching_options(False)

if __name__ == '__main__':
    client = kfp.Client()
    run = client.create_run_from_pipeline_func(
        collected_param_pipeline,
        arguments={},
        enable_caching=False,
    )
