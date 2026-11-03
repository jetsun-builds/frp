"""Black-box tests of both wire protocols, multiplexing and measured traffic."""
import base64
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import threading
import time
import urllib.request

ROOT = Path(__file__).resolve().parent.parent
EXT = '.exe' if os.name == 'nt' else ''
AUTH = 'Basic ' + base64.b64encode(b'test:test').decode()


def port(kind=socket.SOCK_STREAM):
    with socket.socket(socket.AF_INET, kind) as sock:
        sock.bind(('127.0.0.1', 0))
        return sock.getsockname()[1]


def api(admin, path):
    req = urllib.request.Request(f'http://127.0.0.1:{admin}{path}', headers={'Authorization': AUTH})
    with urllib.request.urlopen(req, timeout=2) as response:
        return json.load(response)


def eventually(check, timeout=30):
    deadline = time.monotonic() + timeout
    last = None
    while time.monotonic() < deadline:
        try:
            result = check()
            if result:
                return result
        except (OSError, ValueError, AssertionError) as err:
            last = err
        time.sleep(.2)
    raise AssertionError(f'condition timed out: {last}')


def scenario(wire, mux, udp=True):
    server, sa, ca, remote = (port() for _ in range(4))
    udpport = port(socket.SOCK_DGRAM) if udp else 0
    with tempfile.TemporaryDirectory() as temp, socket.socket() as echo:
        echo.bind(('127.0.0.1', 0))
        echo.listen()
        echo.settimeout(.5)
        stop = threading.Event()

        def serve():
            while not stop.is_set():
                try:
                    conn, _ = echo.accept()
                except socket.timeout:
                    continue
                except OSError:
                    return
                with conn:
                    while data := conn.recv(65536):
                        conn.sendall(data)

        thread = threading.Thread(target=serve, daemon=True)
        thread.start()
        folder = Path(temp)
        server_cfg = folder / 'frps.toml'
        client_cfg = folder / 'frpc.toml'
        server_cfg.write_text(f'''bindAddr = "127.0.0.1"
bindPort = {server}
networkQualityUDPPort = {udpport}
transport.tcpMux = {str(mux).lower()}
auth.token = "quality-test"
webServer.addr = "127.0.0.1"
webServer.port = {sa}
webServer.user = "test"
webServer.password = "test"
''')
        client_cfg.write_text(f'''serverAddr = "127.0.0.1"
serverPort = {server}
clientID = "quality-test"
auth.token = "quality-test"
transport.wireProtocol = "{wire}"
transport.tcpMux = {str(mux).lower()}
webServer.addr = "127.0.0.1"
webServer.port = {ca}
webServer.user = "test"
webServer.password = "test"
[[proxies]]
name = "echo"
type = "tcp"
localIP = "127.0.0.1"
localPort = {echo.getsockname()[1]}
remotePort = {remote}
''')
        procs = []
        with (folder / 'server.log').open('w+') as sl, (folder / 'client.log').open('w+') as cl:
            try:
                procs.append(subprocess.Popen([str(ROOT / 'bin' / ('frps' + EXT)), '-c', str(server_cfg)], stdout=sl, stderr=sl))
                eventually(lambda: api(sa, '/api/serverinfo'))
                procs.append(subprocess.Popen([str(ROOT / 'bin' / ('frpc' + EXT)), '-c', str(client_cfg)], stdout=cl, stderr=cl))

                def measured():
                    q = api(ca, '/api/network-quality')
                    return q if q and q['supported'] and (not udp or q['udpProbeCount'] > 0) else None

                first = eventually(measured)
                assert first['rttMs'] >= 0
                if udp:
                    assert first['udpLossPercent'] == 0, first
                else:
                    assert first['udpLossPercent'] is None and first['udpStatus'] == 'disabled', first
                # Transfer a known payload, verifying both directions and stream counters.
                payload = b'network-quality-test' * 5000
                with socket.create_connection(('127.0.0.1', remote), timeout=5) as conn:
                    conn.sendall(payload)
                    response = bytearray()
                    while len(response) < len(payload):
                        data = conn.recv(65536)
                        assert data, 'proxy closed early'
                        response.extend(data)
                    assert response == payload

                def traffic_reported():
                    q = api(ca, '/api/network-quality')
                    return q if q['bytesSent'] - first['bytesSent'] >= len(payload) and q['bytesReceived'] - first['bytesReceived'] >= len(payload) and q['sendBps'] > 0 and q['receiveBps'] > 0 else None

                eventually(traffic_reported)

                def server_reported():
                    clients = api(sa, '/api/clients')
                    q = clients[0].get('quality') if clients else None
                    return q if q and q['supported'] and q['bytesSent'] >= len(payload) else None

                report = eventually(server_reported)
                assert 'quality_udp_token' not in json.dumps(report)
                assert api(sa, '/api/v2/clients')['data']['items'][0]['quality']['supported']
                procs[-1].terminate()
                procs[-1].wait(timeout=10)
                eventually(lambda: all(not item['online'] and 'quality' not in item for item in api(sa, '/api/clients')))
                print(f'PASS wire={wire} mux={mux} udp={udp}', flush=True)
            except BaseException:
                sl.flush(); cl.flush(); sl.seek(0); cl.seek(0)
                print(sl.read()); print(cl.read())
                raise
            finally:
                stop.set()
                for proc in reversed(procs):
                    if proc.poll() is None:
                        proc.terminate()
                        proc.wait(timeout=10)
                echo.close()
                thread.join(timeout=2)


if __name__ == '__main__':
    for protocol in ('v1', 'v2'):
        for multiplex in (True, False):
            scenario(protocol, multiplex)
    scenario('v1', True, udp=False)
