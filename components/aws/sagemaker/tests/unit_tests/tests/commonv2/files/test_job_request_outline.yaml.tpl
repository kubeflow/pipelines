apiVersion: sagemaker.services.k8s.aws/v1alpha1
kind: TrainingJob
metadata:
  name:
  annotations:
    services.k8s.aws/region: us-west-1
spec:
  spotInstance: False
  maxWaitTime: 1
  inputStr: woof
  inputInt: 1
  inputBool: False