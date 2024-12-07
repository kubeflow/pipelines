from kfp import dsl

@dsl.component(base_image="python:3.11.7", packages_to_install=['qiskit-aer', 'qiskit', 'pylatexenc', 'ipywidgets', 'matplotlib'])
def qpe(n: int):
    from qiskit import QuantumRegister, QuantumCircuit, ClassicalRegister, transpile
    from qiskit_aer import AerSimulator
    from qiskit.visualization import plot_histogram
    from math import pi

    count_no = n #the number of count qubits
    countreg = QuantumRegister(count_no,'count')
    psireg = QuantumRegister(1,'psi')
    creg = ClassicalRegister(count_no,'c')                     
    qc = QuantumCircuit(countreg,psireg,creg)


    for countbit in range(count_no):
      qc.h(countbit)
    qc.x(psireg)
    repeat = 1
    for countbit in range(count_no):
      for r in range(repeat):
        qc.cp(pi/2,countbit,psireg)
      repeat *= 2
    qc.barrier()



    for sbit in range(count_no//2):       #sbit: for swap qubit
      qc.swap(sbit,count_no-sbit-1)  

    for hbit in range(0,count_no,1):      #hbit: for h-gate qubit
      for cbit in range(hbit-1,-1,-1):    #cbit: for count qubit
        qc.cp(-pi/2**(hbit-cbit), cbit, hbit)  
      qc.h(hbit) 
    qc.barrier()


    qc.measure(range(count_no),range(count_no))  
    #display(qc.draw('mpl'))
    
    sim = AerSimulator()
    backend=sim
    new_circuit = transpile(qc, backend)
    job = backend.run(new_circuit)
    result = job.result()
    counts = result.get_counts(qc)
    print("Total counts for qubit states are:",counts)
    plot_histogram(counts)

@dsl.pipeline
def qpe_pipeline(num: int):
    
    qpe_task = qpe(n=num)
    
from kfp import compiler

compiler.Compiler().compile(qpe_pipeline, 'QPE-pipeline.yaml')