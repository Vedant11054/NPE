import sqlite3
from datetime import datetime, timezone

DB = "attacks.db"

def connect():
    conn = sqlite3.connect(DB)
    conn.row_factory = sqlite3.Row
    return conn

def init_db():
    conn = connect()
    conn.execute("""
        CREATE TABLE IF NOT EXISTS events (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            source_ip TEXT NOT NULL,
            source_port INTEGER,
            detected_at TEXT NOT NULL,
            scan_status TEXT NOT NULL,
            open_ports TEXT DEFAULT ''
        )
    """)
    conn.commit()
    conn.close()

def add_event(source_ip, source_port, scan_status="not_scanned", open_ports=""):
    conn = connect()
    cur = conn.execute("""
        INSERT INTO events
        (source_ip, source_port, detected_at, scan_status, open_ports)
        VALUES (?, ?, ?, ?, ?)
    """, (
        source_ip,
        source_port,
        datetime.now(timezone.utc).isoformat(),
        scan_status,
        open_ports
    ))
    event_id = cur.lastrowid
    conn.commit()
    conn.close()
    return event_id

def update_scan(event_id, status, ports):
    conn = connect()
    conn.execute("""
        UPDATE events
        SET scan_status = ?, open_ports = ?
        WHERE id = ?
    """, (status, ",".join(map(str, ports)), event_id))
    conn.commit()
    conn.close()

def get_events():
    conn = connect()
    rows = conn.execute(
        "SELECT * FROM events ORDER BY id DESC"
    ).fetchall()
    conn.close()
    return rows
