package cronTrigger

func main(namespace string, name string) {

	/*
		// Get the ScheduledWorkflow with this namespace/name
		swf, err = c.swfClient.Get(namespace, name)
		if err != nil {
			// Permanent failure.
			// The ScheduledWorkflow may no longer exist, we stop processing and do not retry.
			return false, false, nil,
				wraperror.Wrapf(err, "ScheduledWorkflow (%s) in work queue no longer exists: %v", key, err)
		}

		// If the workflow is not found, we need to create it.
		newWorkflow, err := swf.NewWorkflow(nextScheduledEpoch, nowEpoch)
		createdWorkflow, err := c.workflowClient.Create(swf.Namespace, newWorkflow)
	*/
}
