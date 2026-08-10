# Automated Reverse Port Scanning & Attacker Profiling Lab

A college-project prototype that detects connections to a local honeypot, records the source IP, and performs a **lab-authorized reverse port scan** against that IP.

## Safety / scope

This project is intentionally restricted to IP addresses listed in `config.json` under `authorized_targets`.
Use it only on your own computer/VMs or systems for which you have explicit permission.

## Project flow

Attacker VM -> Honeypot -> Detect source IP -> Authorized reverse scan -> SQLite log -> Flask dashboard

## Requirements

Python 3.10+

Install:

```bash
pip install -r requirements.txt
```

## Configure

Edit `config.json`.

Example for a two-VM lab:

```json
{
  "honeypot_host": "0.0.0.0",
  "honeypot_port": 8080,
  "authorized_targets": ["192.168.56.101"],
  "scan_ports": [21, 22, 23, 25, 53, 80, 110, 139, 143, 443, 445, 3306, 3389, 5432, 8080]
}
```

Replace the target with the IP of your **authorized lab attacker VM**.

## Run

Terminal 1:

```bash
python app.py
```

The honeypot listens on port 8080.

Terminal 2:

```bash
python scanner.py --target 192.168.56.101
```

Or trigger the automatic workflow by connecting to the honeypot from the authorized lab VM:

```bash
nc <VICTIM_IP> 8080
```

The application detects the source IP and automatically scans it if it is on the allowlist.

## Dashboard

Open:

http://127.0.0.1:5000

The dashboard shows detected connections and discovered open ports.

## Files

- `app.py` - honeypot + event detection + dashboard
- `scanner.py` - allowlisted TCP connect scanner
- `database.py` - SQLite database
- `config.json` - lab configuration
- `templates/index.html` - dashboard
- `requirements.txt` - Python dependencies
