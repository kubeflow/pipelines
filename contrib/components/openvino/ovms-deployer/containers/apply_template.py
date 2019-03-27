from jinja2 import Template
import os


f = open("ovms.j2","r")
ovms_template = f.read()
t = Template(ovms_template)
ovms_k8s = t.render(os.environ)
f.close
f = open("ovms.yaml", "w")
f.write(ovms_k8s)
f.close

print(ovms_k8s)