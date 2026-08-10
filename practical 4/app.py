import json
import threading
import socket
from flask import Flask, render_template, jsonify

from database import init_db, add_event, update_scan, get_events
from scanner import scan_target

CONFIG_FILE = "config.json"

with open(CONFIG_FILE, "r", encoding="utf-8") as f:
    CONFIG = json.load(f)

app = Flask(__name__)

def handle_connection(conn, addr):
    source_ip, source_port = addr

    # Record the connection immediately.
    event_id = add_event(
        source_ip,
        source_port,
        scan_status="queued"
    )

    # Only scan explicitly authorized lab targets.
    try:
        ports = scan_target(source_ip)
        update_scan(event_id, "completed", ports)
    except PermissionError:
        update_scan(event_id, "blocked_not_authorized", [])
    except Exception as exc:
        update_scan(event_id, f"error: {type(exc).__name__}", [])

    try:
        conn.sendall(
            b"Connection recorded by the cybersecurity honeypot.\n"
        )
    except OSError:
        pass
    finally:
        conn.close()

def honeypot():
    host = CONFIG["honeypot_host"]
    port = int(CONFIG["honeypot_port"])

    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind((host, port))
    server.listen(20)

    print(f"[HONEYPOT] Listening on {host}:{port}")

    while True:
        conn, addr = server.accept()
        print(f"[DETECTED] Connection from {addr[0]}:{addr[1]}")
        threading.Thread(
            target=handle_connection,
            args=(conn, addr),
            daemon=True
        ).start()

@app.route("/")
def index():
    return render_template("index.html", events=get_events())

@app.route("/api/events")
def api_events():
    return jsonify([dict(row) for row in get_events()])

if __name__ == "__main__":
    init_db()

    threading.Thread(target=honeypot, daemon=True).start()

    print("[DASHBOARD] http://127.0.0.1:5000")
    app.run(host="127.0.0.1", port=5000, debug=False)
