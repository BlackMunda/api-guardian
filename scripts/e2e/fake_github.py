#!/usr/bin/env python3
"""A tiny fake of the GitHub comments API, for local end-to-end rehearsals."""
import json, sys
from http.server import BaseHTTPRequestHandler, HTTPServer

comments = []

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):  # silence default logging
        pass

    def _send(self, code, obj):
        data = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _body(self):
        return json.loads(self.rfile.read(int(self.headers["Content-Length"])))

    def do_GET(self):
        print("GET  ", self.path, flush=True)
        self._send(200, comments)

    def do_POST(self):
        comments.append({"id": len(comments) + 1, "body": self._body()["body"]})
        print("POST ", self.path, "auth:", self.headers.get("Authorization"), flush=True)
        self._send(201, {})

    def do_PATCH(self):
        comments[0]["body"] = self._body()["body"]
        print("PATCH", self.path, flush=True)
        self._send(200, {})

if __name__ == "__main__":
    HTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
