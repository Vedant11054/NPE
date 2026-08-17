import socket
import struct

domain = "google.com"
dns_server = "8.8.8.8"

header = struct.pack("!HHHHHH", 0x1234, 1, 0, 0, 0, 0)

qname = b""

for part in domain.split("."):
    qname += bytes([len(part)]) + part.encode()

qname += b"\x00"

question = qname + struct.pack("!HH", 1, 1)

query = header + question

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.sendto(query, (dns_server, 53))

response, addr = sock.recvfrom(512)

tid, flags, qd, an, ns, ar = struct.unpack("!HHHHHH", response[:12])

print("DNS Server:", addr[0])
print("Transaction ID:", hex(tid))
print("Flags:", hex(flags))
print("Questions:", qd)
print("Answers:", an)
print("Response:", len(response), "bytes")