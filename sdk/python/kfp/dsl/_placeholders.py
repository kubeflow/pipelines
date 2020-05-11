FLAGS = {
			'EXECUTION_ID' : '{{workflow.uid}}-{{pod.name}}',
			'POD_NAME': '{{pod.name}}',
			'RUN_ID' : '{{workflow.uid}}',
			'RUN_EXIT_STATUS' : '{{workflow.status}}',
			'RUN_NAME' : '{{workflow.name}}',
			'RUN_CREATION_TIMESTAMP' : '{{workflow.creationTimestamp}}',
			'RUN_NAMESPACE' : '{{workflow.namespace}}',
			'RUN_DURATION' : '{{workflow.duration}}'
		}

def get_run_information_placeholder(flag_name):
	return FLAGS[flag_name]