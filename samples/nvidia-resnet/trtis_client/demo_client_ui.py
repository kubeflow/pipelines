# Copyright (c) 2019, NVIDIA CORPORATION. All rights reserved.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions
# are met:
#  * Redistributions of source code must retain the above copyright
#    notice, this list of conditions and the following disclaimer.
#  * Redistributions in binary form must reproduce the above copyright
#    notice, this list of conditions and the following disclaimer in the
#    documentation and/or other materials provided with the distribution.
#  * Neither the name of NVIDIA CORPORATION nor the names of its
#    contributors may be used to endorse or promote products derived
#    from this software without specific prior written permission.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS ``AS IS'' AND ANY
# EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
# IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
# PURPOSE ARE DISCLAIMED.  IN NO EVENT SHALL THE COPYRIGHT OWNER OR
# CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
# EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
# PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
# PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY
# OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
# OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


import os
import urllib.request as req
import dash
import dash_core_components as dcc
import dash_html_components as html
from dash.dependencies import Input, Output
from dash.exceptions import PreventUpdate


external_stylesheets = ['https://codepen.io/chriddyp/pen/bWLwgP.css']

app = dash.Dash(__name__, external_stylesheets=external_stylesheets)

app.layout = html.Div([
    html.H3('Client UI'),
    html.Div('Image URL:'),
    dcc.Input(
        id='url-input',
        placeholder='Enter the image url...',
        type='text',
        value='',
        style={'display': 'block'}
    ),
    html.Img(
        id='web-image',
        width='500'
    ),
    html.Div('Prediction:'),
    html.Div(
        id='prediction-div',
        style={'whiteSpace': 'pre-wrap'}
    )
])


@app.callback(
    Output('web-image', 'src'),
    [Input('url-input', 'value')])
def display_image(url):
    return url


@app.callback(
    Output('prediction-div', 'children'),
    [Input('url-input', 'value')])
def get_prediction(url):
    if not url:
        raise PreventUpdate
    img_path = 'sample.jpg'
    result_path = 'result.txt'
    script = '/workspace/src/clients/python/updated_image_client.py'
    model_name = 'resnet_graphdef'
    scaling_style = 'RESNET'
    req.urlretrieve(url, img_path)
    if os.path.exists(result_path):
        os.remove(result_path)
    cmd = 'python3 %s -m %s -s %s %s >> %s' % (
        script, model_name, scaling_style, img_path, result_path)
    os.system(cmd)
    with open(result_path, 'r') as f:
        result = f.read()
    return result


if __name__ == '__main__':
    app.run_server(debug=True, host='0.0.0.0', port=8050)