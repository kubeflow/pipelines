## Use cronjob

# Job
Should contain reference to:
- cronJob resource
- pipeline itself or we keep it as it is currently and store it completely
An update is an update of the job in the database. 
- When we enabled the job we create a new cronJob resource. 

User story: 
- To update an job, the user will just point to a new pipeline or version or update the cron expression. Updating the database in terms of pipeline or argument should be enough. 
- When a job is disabled, no cron expressions is runing. We could also keep it runing at all times .... and just check if it should run until the job is deleted. 

When a job is created:
1. The job is created in the database 
2. A Cron job is created in the namespace. 
3. When the cronjob rung(slim go container)
    1. Pull the job info from the database
    2. Check if pipeline is enabled
    3. Run the job, add reference as part of the name to the job and in the experiment of interest. 
4. When the job is updated the database is updated only. If we change time we need to patch the cronjob. This should be easy enough. 