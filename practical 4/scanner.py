import argparse
import ipaddress
import json
import socket

CONFIG_FILE = "config.json"

def load_config():
    with open(CONFIG_FILE, "r", encoding="utf-8") as f:
        return json.load(f)

def is_authorized(target, authorized):
    try:
        target_ip = ipaddress.ip_address(target)
    except ValueError:
        return False

    for entry in authorized:
        try:
            if "/" in entry:
                if target_ip in ipaddress.ip_network(entry, strict=False):
                    return True
            elif target_ip == ipaddress.ip_address(entry):
                return True
        except ValueError:
            continue
    return False

def scan_target(target):
    config = load_config()

    if not is_authorized(target, config["authorized_targets"]):
        raise PermissionError(
            f"{target} is not in config.json authorized_targets"
        )

    ports = config["scan_ports"]
    timeout = float(config.get("scan_timeout", 0.35))
    open_ports = []

    for port in ports:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(timeout)
        try:
            result = sock.connect_ex((target, int(port)))
            if result == 0:
                open_ports.append(int(port))
        except (OSError, ValueError):
            pass
        finally:
            sock.close()

    return open_ports

if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Authorized lab TCP port scanner"
    )
    parser.add_argument("--target", required=True)
    args = parser.parse_args()

    try:
        result = scan_target(args.target)
        print(f"Target: {args.target}")
        print("Open ports:", result if result else "None found")
    except PermissionError as e:
        print("BLOCKED:", e)
        raise SystemExit(2)
