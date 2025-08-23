
import kfp  
import kfp.kubernetes  
from kfp import dsl  
from kfp.dsl import Artifact, Input, Output  
  
  
@dsl.component()  
def fail(model_id: str):  
    import sys  
    print(model_id)  
    sys.exit(1)  
  
@dsl.component()  
def hello_world():  
    print("hellow_world")  
  
@dsl.pipeline(name="Pipeline", description="Pipeline")  
def export_model():  
    # For the iteration_index execution, total_dag_tasks is always 2  
    # because this value is generated from the # of tasks in the component dag (generated at sdk compile time)    # however parallelFor can be a dynamic number and thus likely    # needs to match iteration_count (generated at runtime)    	  
    with dsl.ParallelFor(items=['1', '2', '3']) as model_id:  
        hello_task = hello_world().set_caching_options(enable_caching=False)  
        fail_task = fail(model_id=model_id).set_caching_options(enable_caching=False)  
        fail_task.after(hello_task)  
  
if __name__ == "__main__":  
    kfp.compiler.Compiler().compile(export_model, "loops.yaml")
