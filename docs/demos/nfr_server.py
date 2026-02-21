# Helper server for NFR demo: serves a small JSON response on all paths.
import http.server
import json
import sys
import threading
import subprocess


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"id": 42}).encode())

    def log_message(self, *args):
        pass


def run_verify(contract_yaml, extra_msg=""):
    server = http.server.HTTPServer(("127.0.0.1", 0), Handler)
    port = server.server_address[1]
    t = threading.Thread(target=server.serve_forever, daemon=True)
    t.start()
    r = subprocess.run(
        ["./accord", "verify", "--provider-url", f"http://127.0.0.1:{port}", "/dev/stdin"],
        input=contract_yaml.encode(),
        capture_output=True,
    )
    sys.stdout.buffer.write(r.stdout)
    if extra_msg:
        print(extra_msg.replace("EXIT", str(r.returncode)))
    server.shutdown()
    sys.exit(r.returncode)
