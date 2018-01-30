# Example Dags, Templates, etc.

## dags:
**zip_dag_1.zip** **zip_dag_2.zip** :

This example tests how Airflow handles DAG dependencies which exist inside the .zip package, specifically, whether the dependencies conflicts with eath other when two .zip packages have same depdencies.

In this case, both Dag packages contain 
```python
class MyFirstOperator(BaseOperator): {...}
```
dependency with differnet implementation. By putting both packages under Airflow dags/ directory, Airflow is able to pick up the right dependency.


**print_pythonpath.py** :

  Example to retrieve the PYTHONPATH in the context of a Airflow worker. To better illustrate the difference, it assumes **zip_dag_1.zip** **zip_dag_2.zip** are in the same Airflow dags/ directory.


  By triggering the DAG `airflow trigger_dag print_pythonpath`, it prints following:
```
  /usr/local/bin  
  /usr/lib/python2.7  
  /usr/lib/python2.7/plat-x86_64-linux-gnu  
  /usr/lib/python2.7/lib-tk  
  /usr/lib/python2.7/lib-old  
  /usr/lib/python2.7/lib-dynload  
  /usr/local/lib/python2.7/dist-packages  
  /usr/lib/python2.7/dist-packages  
  /usr/lib/python2.7/dist-packages/PILcompat  
  /root/airflow/config  
  /root/airflow/dags  
  /root/airflow/plugins  
```
Which doesn't load zip packages to its path. 


P.S. In contrast, if I do a operator test `airflow test print_pythonpath python_task 2017-03-18T18:00:00.0`, it will look for python dependencies from the sibling zip packages:
```
  /root/airflow/dags/zip_dag_2.zip  
  /root/airflow/dags/zip_dag_1.zip  
  /usr/local/bin  
  /usr/lib/python2.7  
  /usr/lib/python2.7/plat-x86_64-linux-gnu  
  /usr/lib/python2.7/lib-tk  
  /usr/lib/python2.7/lib-old  
  /usr/lib/python2.7/lib-dynload  
  /usr/local/lib/python2.7/dist-packages  
  /usr/lib/python2.7/dist-packages  
  /usr/lib/python2.7/dist-packages/PILcompat  
  /root/airflow/config  
  /root/airflow/dags  
  /root/airflow/plugins  
```
TODO(yangpa): Maybe it's good idea to dig into code a little bit to better understand why.