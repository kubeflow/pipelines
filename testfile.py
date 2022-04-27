from kfp import client

c = client.Client(
    'https://4176b1d345e35747-dot-us-central1.pipelines.googleusercontent.com/')

res = c.get_kfp_healthz()
print(res)
