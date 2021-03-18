# KFP Driver

## Responsibility

* Read KFP task's context from MLMD.
* Init MLMD execution.
* Resolve input artifacts and parameters and output them through argo parameters.

## Pseudo Code

```go
func driver(parentExecutionId int, taskSpec Spec) {
    mlmdClient := newMlmdClient()
    parentExecution := mlmdClient.GetExecutionById(parentExecutionId)
    parentParameters := parentExecution.GetParameters()
    for _, paramName, paramSource := taskSpec.Inputs.Parameters {
        if paramSource == runtimeValueOrParameter {
            ioutil.write(
                parentExecution.GetCustomProperty("input:"+paramName),
                "/kfp/parameters/" + paramName
            )
        }
    }
}
```
