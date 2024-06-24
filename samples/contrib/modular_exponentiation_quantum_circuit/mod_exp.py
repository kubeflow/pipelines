from kfp import dsl
@dsl.component(base_image="python:3.11.7", packages_to_install=['qiskit','matplotlib','pylatexenc'])
def modular_exponentiation(power_num: int) -> dsl.Artifact:
    from qiskit import QuantumRegister, QuantumCircuit
    def mod_func(a, power, show=False):
        
        assert a in [2,4,7,8,11,13], 'Invalid value of argument a:' + str(a)
        qrt = QuantumRegister(4, 'target')
        U = QuantumCircuit(qrt)        
        for i in range(power):
            if a in [2,13]:
                U.swap(0,1)
                U.swap(1,2)
                U.swap(2,3)
            if a in [7,8]:
                U.swap(2,3)
                U.swap(1,2)
                U.swap(0,1)
            if a in [4, 11]:
                U.swap(1,3)
                U.swap(0,2)
            if a in [7,11,13]:
                for j in range(4):
                    U.x(j)
        if show:
            print('Below is the circuit of U of ' + f'"{a}^{power} mod 15":') 
        U = U.to_gate()
        U.name = f'{a}^{power} mod 15'
        C_U = U.control()
        return C_U

    k = []
    power_arg = power_num
    for a_arg in [2,4,7,8,11,13]:
        qrc = QuantumRegister(1, 'control')
        qrt = QuantumRegister(4, 'target')
        qc = QuantumCircuit(qrc, qrt)
        qc.append(mod_func(a_arg, power_arg, show=True), [0,1,2,3,4])
        print('Below is the circuit of controlled U of ' + f'"{a_arg}^{power_arg} mod 15":')
        k.append(qc.draw('mpl'))
    return k
@dsl.pipeline
def modular_exponentiation_pipeline(num: int):
    
    modular_exponentiation_task = modular_exponentiation(power_num=num)

from kfp import compiler

compiler.Compiler().compile(modular_exponentiation_pipeline, 'modular_exponentiation_pipeline.yaml')