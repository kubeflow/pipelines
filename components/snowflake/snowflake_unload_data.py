"""
This is a KFP component doing "unload data to GCS bucket" operation
  from the Snowflake database.
"""
from kfp import compiler
from kfp.dsl import component

@component(
    base_image="python:3.11",
    packages_to_install=["snowflake-connector-python:3.12.3"]
)
def snowflake_unload_op(
    output_gcs_path: str,
    sf_storage_integration: str,
    query: str,
    sf_user: str,
    sf_password: str,
    sf_account: str,
    sf_warehouse: str,
    sf_database: str
    ) -> str:
    """
    Run COPY in snowflake to unload data to GCS bucket.

    output_gcs_path: the location to land the data
    sf_storage_integration: the Snowflake Storage Integration name
    query: the query to execute in the COPY command
    sf_user: the snowflake username
    sf_password: the snowflake password
    sf_warehouse: the snowflake warehouse name
    sf_database: the database to use
    """
    import snowflake.connector
    conn = snowflake.connector.connect(user=sf_user,
                                       password=sf_password,
                                       account=sf_account,
                                       role="ACCOUNTADMIN")

    conn.cursor().execute(f"USE WAREHOUSE {sf_warehouse};")
    conn.cursor().execute(f"USE DATABASE {sf_database};")
    result = conn.cursor().execute(f"""
    COPY INTO 'gcs://{output_gcs_path}'
    FROM ({query})
    FILE_FORMAT = (TYPE = CSV COMPRESSION=NONE)
    STORAGE_INTEGRATION = {sf_storage_integration}
    HEADER = TRUE
    """)
    _ = result.fetchall()
    if output_gcs_path.endswith("/"):
        return output_gcs_path + "data_0_0_0.csv"
    else:
        return output_gcs_path

if __name__ == "__main__":
    compiler.Compiler().compile(pipeline_func=snowflake_unload_op,
                                package_path="component.yaml")
