# Copyright 2025 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import http.server
import os
import socketserver
import webbrowser
from typing import Optional

import click

BANNER_HTML = """
<style>
    #kfp-local-banner-top {
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        border-top: 4px solid #1a73e8;
    }
    #kfp-local-banner-top:hover {
        background: rgba(255, 255, 255, 1) !important;
        box-shadow: 0 8px 20px rgba(0,0,0,0.12) !important;
        transform: translateY(2px);
    }
</style>
<div id="kfp-local-banner-top" style="background: rgba(255, 255, 255, 0.96); backdrop-filter: blur(12px); color: #3c4043; padding: 14px 28px; text-align: center; font-family: 'Google Sans', Roboto, sans-serif; font-size: 14px; border-bottom: 1px solid rgba(0,0,0,0.1); position: sticky; top: 0; z-index: 9999; display: flex; align-items: center; justify-content: center; gap: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.08); cursor: default;">
    <div style="display: flex; align-items: center; gap: 10px; color: #1a73e8; font-weight: 500;">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor" style="flex-shrink: 0;"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
        <span style="font-size: 15px; letter-spacing: 0.2px;">Local Mode</span>
    </div>
    <div style="width: 1px; height: 24px; background: #e0e0e0;"></div>
    <span style="color: #5f6368; line-height: 1.5; font-size: 14px;">Serving static assets. Deploy to a cluster for execution functionality.</span>
</div>
"""

class LocalUIHandler(http.server.SimpleHTTPRequestHandler):

    def log_message(self, format, *args):
        return

    def do_GET(self):
        if self.path == '/' or self.path == '/index.html':
            index_path = os.path.join(self.directory, 'index.html')
            if os.path.exists(index_path):
                self.send_response(200)
                self.send_header('Content-type', 'text/html')
                self.end_headers()
                with open(index_path, 'r') as f:
                    content = f.read()
                
                if '<body>' in content:
                    content = content.replace('<body>', f'<body>{BANNER_HTML}')
                elif '<body ' in content:
                    content = content.replace('<body ', f'<body {BANNER_HTML}', 1)
                
                self.wfile.write(content.encode('utf-8'))
                return
        
        return super().do_GET()


def find_assets() -> Optional[str]:
    possible_paths = [
        os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..', '..', '..', 'frontend', 'build')),
        os.path.join(os.path.dirname(__file__), 'uistatic'),
    ]
    
    for path in possible_paths:
        if os.path.exists(os.path.join(path, 'index.html')):
            return path
    return None


@click.command()
@click.option('--port', default=5000, help='Port to serve the UI on.')
@click.option('--no-browser', is_flag=True, help='Do not open the browser automatically.')
def ui(port: int, no_browser: bool):
    asset_path = find_assets()
    
    if not asset_path:
        raise click.ClickException(
            'Could not find KFP UI assets. Please ensure the frontend is built '
            'or the ui-static assets are included in the package.'
        )
    
    click.echo(f'Serving KFP UI from {asset_path}')
    click.echo(f'Access the UI at http://localhost:{port}')
    
    handler = lambda *args, **kwargs: LocalUIHandler(*args, directory=asset_path, **kwargs)
    
    socketserver.TCPServer.allow_reuse_address = True
    try:
        with socketserver.TCPServer(("", port), handler) as httpd:
            if not no_browser:
                webbrowser.open(f'http://localhost:{port}')
            click.echo('Press Ctrl+C to stop the server.')
            httpd.serve_forever()
    except OSError as e:
        if e.errno == 98:
            raise click.ClickException(
                f'Port {port} is already in use. Please use a different port with --port <port> '
                'or stop the process currently using this port.'
            )
        raise e
    except KeyboardInterrupt:
        click.echo('\nStopping KFP UI server...')
