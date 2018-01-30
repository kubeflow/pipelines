# Example Dags, Templates, etc.

## dags:
**zip_dag_1.zip** **zip_dag_2.zip** :

Both Dag packages contain 
```python
class MyFirstOperator(BaseOperator): {...}
```
  dependency with differnet implementation. By putting both packages under Airflow /dag dir, Airflow is able to pick up the right dependency without collision.


**print_pythonpath.py** :

  Print the paths to load python dependencies.
  By running `airflow trigger_dag print_pythonpath`, it print following from my container:

  /usr/local/bin'  
  /usr/lib/python2.7'  
  /usr/lib/python2.7/plat-x86_64-linux-gnu'  
  /usr/lib/python2.7/lib-tk'  
  /usr/lib/python2.7/lib-old'  
  /usr/lib/python2.7/lib-dynload'  
  /usr/local/lib/python2.7/dist-packages'  
  /usr/lib/python2.7/dist-packages'  
  /usr/lib/python2.7/dist-packages/PILcompat'  
  /root/airflow/config'  
  /root/airflow/dags'  
  /root/airflow/plugins'  


P.S. If I run test command `airflow test print_pythonpath python_task 2017-03-18T18:00:00.0`, it will include zip packages:

  /root/airflow/dags/zip_dag_2.zip'  
  /root/airflow/dags/zip_dag_1.zip'  
  /usr/local/bin'  
  /usr/lib/python2.7'  
  /usr/lib/python2.7/plat-x86_64-linux-gnu'  
  /usr/lib/python2.7/lib-tk'  
  /usr/lib/python2.7/lib-old'  
  /usr/lib/python2.7/lib-dynload'  
  /usr/local/lib/python2.7/dist-packages'  
  /usr/lib/python2.7/dist-packages'  
  /usr/lib/python2.7/dist-packages/PILcompat'  
  /root/airflow/config'  
  /root/airflow/dags'  
  /root/airflow/plugins'  