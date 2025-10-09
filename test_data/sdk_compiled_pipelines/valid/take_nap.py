from kfp import compiler, dsl

common_base_image = (
    "registry.redhat.io/ubi8/python-39@sha256:3523b184212e1f2243e76d8094ab52b01ea3015471471290d011625e1763af61"
)


@dsl.component(base_image=common_base_image)
def take_nap(naptime_secs: int) -> str:
    """Sleeps for secs"""
    from time import sleep  # noqa: PLC0415

    print(f"Sleeping for {naptime_secs} seconds: Zzzzzz ...")
    sleep(naptime_secs)
    return "I'm awake now. Did I snore?"


@dsl.component(base_image=common_base_image)
def wake_up(message: str):
    """Wakes up from nap printing a message"""
    print(message)


@dsl.pipeline(name="take-nap-pipeline", description="Pipeline that sleeps for 15 mins (900 secs)")
def take_nap_pipeline(naptime_secs: int = 900):
    take_nap_task = take_nap(naptime_secs=naptime_secs).set_caching_options(False)
    wake_up(message=take_nap_task.output).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(take_nap_pipeline, package_path=__file__.replace(".py", "_compiled.yaml"))
