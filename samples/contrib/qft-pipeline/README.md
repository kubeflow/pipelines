# QFT-on-kubeflow
A sample code of Quantum Field Theory (QFT) with kubeflow pipeline.


Installation
---
```
pip install qiskit==0.46.0
pip install qiskit ipywidgets
pip install pylatexenc
```

Define funciton to build n-qubit QFT quantum circuit
---
```
def qft(n):
  ar = QuantumRegister(n,'a')
  qc = QuantumCircuit(ar)
  for hbit in range(n-1,-1,-1):
    qc.h(hbit) 
    for cbit in range(hbit):
      qc.cp(pi/2**(hbit-cbit), cbit, hbit)  
  for bit in range(n//2):
    qc.swap(bit,n-bit-1)  
  return qc  
  ```

Display the QFT circuit
---
```
for i in range(1,5):
  print('Below is the QFT circuit of',i,'qubit(s):') 
  display(qft(i).draw('mpl'))
```

# Pipeline
see the qft_pipe.ipynb file.

Build component
---
```
from kfp import dsl

@dsl.component(base_image="python:3.11.7", packages_to_install=['qiskit-aer', 'qiskit==0.46.0', 'pylatexenc', 'ipywidgets', 'matplotlib'])
```











