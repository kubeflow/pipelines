from kfp import dsl
from typing import Tuple
@dsl.component(base_image="python:3.11.7")
def shor_alg(num:int)-> dsl.Artifact:
    from random import randint
    from math import gcd
    
    def period_finding(a: int, N: int) -> int:
        for r in range(1, N):
            if (a**r) % N == 1:
                return r
    
    while True:
        a = randint(2, num - 1)
        g = gcd(a, num)
    
        if g != 1:
            p = g
            q = num // g
            return p, q
        else:
            r = period_finding(a, num)
            if r % 2 != 0:
                continue
            elif a**(r // 2) % num == -1 % num:
                continue
            else:
                p = gcd(a**(r // 2) + 1, num)
                if p == 1 or p == num:
                    p = gcd(a**(r // 2) - 1, num)
                q = num // p
                return p, q

@dsl.pipeline
def shor_alg_pipeline(num: int):
    
    shor_alg_task = shor_alg(num=num)

from kfp import compiler

compiler.Compiler().compile(shor_alg_pipeline, 'shor_alg_pipeline.yaml')