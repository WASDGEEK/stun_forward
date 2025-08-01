#!/usr/bin/env python3
import http.server
import socketserver
import datetime

class TestHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-type', 'text/html')
        self.end_headers()
        
        response = f"""
        <html>
        <body>
        <h1>STUN转发测试服务器</h1>
        <p>时间: {datetime.datetime.now()}</p>
        <p>请求路径: {self.path}</p>
        <p>客户端地址: {self.client_address}</p>
        <p>🎉 数据转发成功!</p>
        </body>
        </html>
        """
        self.wfile.write(response.encode())

if __name__ == "__main__":
    PORT = 5201
    Handler = TestHandler
    
    with socketserver.TCPServer(("", PORT), Handler) as httpd:
        print(f"测试服务器启动在端口 {PORT}")
        print(f"访问 http://localhost:{PORT} 进行测试")
        httpd.serve_forever()