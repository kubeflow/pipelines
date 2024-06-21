import os
import subprocess

def python3_version():
    return subprocess.check_call(["python3", "--version"])

def which(command):
    return subprocess.check_call(["which", command])

def pip3_install_requirements():
    return subprocess.check_call(["pip3", "install", "-r", "requirements.txt",
                                  "--user"])

def pip3_install_kfp_sever_api():
    return subprocess.check_call(["pip3", "install",
                                  "git+https://github.com/kubeflow/pipelines.git@1.8.19#subdirectory=backend/api/python_http_client",
                                  "--user"])
    
def pip3_install_kfp():
    return subprocess.check_call(["pip3", "install", "kfp==1.7", "--user"])

python3_version()
pip3_install_requirements()
#pip3_install_kfp_sever_api()
pip3_install_kfp()

print("done")
