#!/usr/bin/env python3
"""
Errorium Frontend Dev Server
Lightweight, zero-dependency development server for the Errorium SPA.
- Serves static files on http://localhost:3000
- Implements SPA routing fallback to /index.html
- Proxies /api/* and /health to Gateway (:8000)
"""

import http.server
import mimetypes
import os
import sys
import urllib.error
import urllib.request

PORT = int(os.environ.get("PORT", 3000))
GATEWAY_URL = os.environ.get("GATEWAY_API_URL", "http://localhost:8000").rstrip("/")
STATIC_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "static"))

# Ensure common MIME types are properly recognized
mimetypes.add_type("text/javascript", ".js")
mimetypes.add_type("text/css", ".css")
mimetypes.add_type("image/svg+xml", ".svg")
mimetypes.add_type("image/png", ".png")
mimetypes.add_type("application/json", ".json")

class ErroriumDevHandler(http.server.BaseHTTPRequestHandler):
    def do_HEAD(self):
        self._handle_request()

    def do_GET(self):
        self._handle_request()

    def do_POST(self):
        self._handle_request()

    def do_PUT(self):
        self._handle_request()

    def do_DELETE(self):
        self._handle_request()

    def do_OPTIONS(self):
        self._handle_request()

    def _handle_request(self):
        # 1. Check if request should be proxied to Gateway
        if self.path.startswith("/api/") or self.path.startswith("/health"):
            self._proxy_to_gateway()
            return

        # 2. Serve static SPA files
        self._serve_static()

    def _proxy_to_gateway(self):
        target_url = f"{GATEWAY_URL}{self.path}"
        content_length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_length) if content_length > 0 else None

        req_headers = {k: v for k, v in self.headers.items() if k.lower() != "host"}

        try:
            req = urllib.request.Request(
                target_url,
                data=body,
                headers=req_headers,
                method=self.command
            )
            with urllib.request.urlopen(req, timeout=5) as resp:
                self.send_response(resp.status)
                for header, val in resp.headers.items():
                    if header.lower() not in ["transfer-encoding", "content-length"]:
                        self.send_header(header, val)
                content = resp.read()
                self.send_header("Content-Length", str(len(content)))
                self.end_headers()
                self.wfile.write(content)
        except urllib.error.HTTPError as err:
            self.send_response(err.code)
            for header, val in err.headers.items():
                if header.lower() not in ["transfer-encoding", "content-length"]:
                    self.send_header(header, val)
            err_content = err.read()
            self.send_header("Content-Length", str(len(err_content)))
            self.end_headers()
            self.wfile.write(err_content)
        except Exception as ex:
            self.send_response(502)
            self.send_header("Content-Type", "application/json")
            error_payload = f'{{"error": "Gateway proxy error: {str(ex)}"}}'.encode("utf-8")
            self.send_header("Content-Length", str(len(error_payload)))
            self.end_headers()
            self.wfile.write(error_payload)

    def _serve_static(self):
        clean_path = self.path.split("?")[0].lstrip("/")
        if clean_path == "":
            clean_path = "index.html"

        file_path = os.path.join(STATIC_DIR, clean_path)

        # Security check: prevent directory traversal
        if not os.path.abspath(file_path).startswith(STATIC_DIR):
            self.send_error(403, "Access denied")
            return

        # SPA fallback: if file does not exist, serve index.html
        if not os.path.exists(file_path) or os.path.isdir(file_path):
            file_path = os.path.join(STATIC_DIR, "index.html")

        mime_type, _ = mimetypes.guess_type(file_path)
        if not mime_type:
            mime_type = "application/octet-stream"

        try:
            with open(file_path, "rb") as f:
                content = f.read()

            self.send_response(200)
            self.send_header("Content-Type", mime_type)
            self.send_header("Content-Length", str(len(content)))
            # Prevent caching during development
            self.send_header("Cache-Control", "no-cache, no-store, must-revalidate")
            self.end_headers()
            if self.command != "HEAD":
                self.wfile.write(content)
        except Exception as ex:
            self.send_error(500, f"Error reading file: {ex}")

    def log_message(self, format, *args):
        # Format log output cleanly
        sys.stderr.write(f"[DevServer] {self.address_string()} - {format % args}\n")

def run():
    server_address = ("", PORT)
    httpd = http.server.ThreadingHTTPServer(server_address, ErroriumDevHandler)
    banner = f"""
======================================================================
  Errorium SPA Dev Server
  - Local URL:      http://localhost:{PORT}
  - Gateway Proxy:  {GATEWAY_URL}
  - Static Assets:  {STATIC_DIR}
======================================================================
Press Ctrl+C to stop.
"""
    print(banner)
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\nStopping Dev Server...")
        httpd.server_close()

if __name__ == "__main__":
    run()
