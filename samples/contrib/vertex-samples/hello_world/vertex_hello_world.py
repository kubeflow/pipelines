from kfp.v2 import dsl

@dsl.component()
def hello_world(text: str) -> str:
    print(text)
    return text


@dsl.pipeline(name='hello-world', description='A hello world example on Vertex')
def pipeline_hello_world():
    hello_world(text='hi, Vertex')


if __name__ == "__main__":
    from kfp.v2 import compiler
    compiler.Compiler().compile(pipeline_func=pipeline_hello_world, package_path='vertex_hello_world.json')
