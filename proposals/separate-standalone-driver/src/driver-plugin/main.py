from fastapi import FastAPI, Request
import uvicorn
from contextlib import asynccontextmanager
import aiohttp


@asynccontextmanager
async def lifespan(app: FastAPI):
    app.state.session = aiohttp.ClientSession()
    yield
    await app.state.session.close()
app = FastAPI(lifespan=lifespan)


async def call_driver(url, request: Request):
    session: aiohttp.ClientSession = request.app.state.session
    body = await request.json()
    payload = body.get("template", {}).get("plugin", {}).get("driver-plugin", {}).get("args", {})
    async with session.post(url, json=payload) as response:
        if response.status != 200:
            text = await response.text()
            raise Exception(f"driver call failed with status: {response.status} error: {text}")
        content_type = response.headers.get("Content-Type", "")
        if "application/json" not in content_type:
            text = await response.text()
            raise Exception(f"driver returns unexpected Content-Type: {content_type}, response: {text}")
        return await response.json()


@app.post("/api/v1/template.execute")
async def execute_plugin(request: Request):
    response = await call_driver("http://ml-pipeline-kfp-driver.kubeflow.svc:2948/api/v1/execute", request)
    return response


if __name__ == "__main__":
    uvicorn.run("main:app", host="0.0.0.0", port=2948, reload=True)