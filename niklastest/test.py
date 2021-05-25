import kfp as kfp

@kfp.components.func_to_container_op
def print_func(param: str):
    print(str(param))
    return

@kfp.dsl.pipeline(name='pipeline')
def pipeline(param: str):
    print_func(param)
    return

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(pipeline, __file__ + ".zip")
