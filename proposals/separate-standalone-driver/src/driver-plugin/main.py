from fastapi import FastAPI, Request
import uvicorn
import aiohttp

app = FastAPI()


async def call_driver(url, payload):
    async with aiohttp.ClientSession() as session:
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
    body = await request.json()
    payload = body.get("template", {}).get("plugin", {}).get("driver-plugin", {}).get("args", {})
    print("request payload:" + str(payload))
    response = await call_driver("http://ml-pipeline-kfp-driver.kubeflow.svc.cluster.local:2948/api/v1/execute", payload)
    print("response:", response)
    return response


if __name__ == "__main__":
    uvicorn.run("main:app", host="0.0.0.0", port=2948, reload=True)
