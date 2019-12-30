import kfp
import argparse

parser = argparse.ArgumentParser()
parser.add_argument('--version_name', help='Required. Name of the new version. Must be unique.', type=str)
parser.add_argument('--package_url', help='Required. pipeline package url', type=str)
parser.add_argument('--pipeline_id', help = 'Required. pipeline id',type=str)
parser.add_argument('--bucket_name', help='Required. gs bucket to store files', type=str)
parser.add_argument('--gcr_address', help='Required. Cloud registry address. For example, gcr.io/my-project', type=str)
parser.add_argument('--host', help='Host address of kfp.Client. Will be get from cluster automatically', type=str, default='')
parser.add_argument('--run_name', help='name of the new run.', type=str, default='')
parser.add_argument('--experiment_id', help = 'experiment id',type=str)
parser.add_argument('--code_source_url', help = 'url of source code', type=str, default='')
args = parser.parse_args()

if args.host:
    client = kfp.Client(host=args.host)
else:
    client = kfp.Client()

print('your client configuration is :{}'.format(client.pipelines.api_client.configuration.__dict__))
print('Now in create_pipeline_version_and_run.py...')
print('your api_client host is:')
print(client.pipelines.api_client.configuration.host)

#create version
version_body = {"name": args.version_name, \
"code_source_url": args.code_source_url, \
"package_url": {"pipeline_url": args.package_url}, \
"resource_references": [{"key": {"id": args.pipeline_id, "type":3}, "relationship":1}]}
print('version body: {}'.format(version_body))
response = client.pipelines.create_pipeline_version(version_body)

print('args are: {}'.format(args))
print('Now start to create a run...')
version_id = response.id
# create run
print('version response: {}'.format(response))
run_name = args.run_name if args.run_name else 'run' + version_id
resource_references = [{"key": {"id": version_id, "type":4}, "relationship":2}]
if args.experiment_id:
    resource_references.append({"key": {"id": args.experiment_id, "type":1}, "relationship": 1})
run_body={"name":run_name,
          "pipeline_spec":{"parameters": [{"name": "bucket_name", "value": args.bucket_name}, {"name": "gcr_address", "value": args.gcr_address}]},
          "resource_references": resource_references}
print('run body is :{}'.format(run_body))
try:
    client.runs.create_run(run_body)
except:
    print('Error Creating Run...')



    
