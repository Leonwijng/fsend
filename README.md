# fsend — Quick file sharing

fsend is a compact, cross-platform Go application for transferring files between devices. It provides:

- A TCP-based file server that stores files per-client using UUIDs
- A client that runs as a GUI (Gio) or as a CLI (interactive menu)

This README explains what fsend is, which protocols it uses, and how it works.

---

## What is fsend?

fsend is a minimal file sharing system intended for quick peer-to-peer-style transfers via a small central server.

- The server accepts TCP connections from clients and stores files in per-UUID directories under `server/files/{uuid}/`.
- Each client has a persistent UUID that is registered with the server on connect.
- Clients can upload files to their own storage, request a list of their stored files, download files, and send a file directly into another client's UUID storage.

The project is written in Go and uses Gio for the GUI. There is a second client folder (`client2/`) useful for running two client instances for testing.

---

## Protocols used

fsend uses the following protocols and encodings:

- Transport: TCP (server listens on TCP port `:3002` by default)
- Application protocol: a custom compact binary protocol (opcode-driven)
- Endianness: Little Endian for all integer fields

The application protocol is very small and binary. Each request begins with a single-byte opcode (uint8). The code defines numeric opcode values as follows (from the source):

- 0 = putfile      — upload a file to the client's own storage
- 1 = listFiles    — request a list of files in the client's storage
- 2 = streamFile   — request/download a file (server streams bytes)
- 3 = ping         — ping server (server replies with text "pong")
- 4 = bye          — close connection
- 5 = register     — register client UUID with the server
- 6 = sendToUUID   — send a file to another client's UUID (server writes file into target UUID dir)

Message field notes (high-level):

- Filenames are prefixed with a single byte length (uint8). Filenames must be <= 255 bytes.
- File sizes are encoded as uint64 (Little Endian).
- Buffer size (optional) is encoded as uint32.
- UUIDs are sent as a length-prefixed byte sequence (uint8 length then bytes).

Example high-level formats (not exhaustive):

- register: [opcode=5][uuidLen:uint8][uuid:bytes]
- putfile:  [opcode=0][fnameLen:uint8][fname:bytes][fsize:uint64][bufSize:uint32][file bytes...]
- sendToUUID: [opcode=6][targetUUIDLen:uint8][targetUUID:bytes][fnameLen:uint8][fname:bytes][fsize:uint64][file bytes...]
- listFiles: [opcode=1]
- streamFile: [opcode=2][fnameLen:uint8][fname:bytes]
- ping: [opcode=3]
- bye: [opcode=4]

All multi-byte integers use Little Endian encoding (see `binary.Read` / `binary.Write` in the source).

---

## How fsend works

This section explains the typical flows and server behavior.

1) Client start and registration

- On first run the client generates a UUID (persisted locally) and connects to the server.
- The client immediately sends a `register` message containing its UUID. The server ensures a directory exists for that UUID under `server/files/{uuid}/`.

2) Upload (putfile)

- The client sends opcode `putfile` with the filename, file size, optional buffer size, then streams the raw file bytes.
- The server reads the filename and file size and saves the incoming bytes into `server/files/{client-uuid}/{filename}`.
- The server handles each client connection in its own goroutine so multiple uploads can happen concurrently.

3) List files

- The client sends opcode `listFiles`.
- The server reads the directory for that client's UUID and returns a file list (implementation detail: see `handleListFiles` in `server/files.go`).

4) Download (streamFile)

- The client sends opcode `streamFile` + filename length + filename.
- The server opens the file from the client's UUID directory and streams the bytes back over the same TCP connection.

5) Send to another UUID (sendToUUID)

- The client sends opcode `sendToUUID`, then the target UUID, filename and file bytes.
- The server writes the file into `server/files/{target-uuid}/{filename}` so the target user can later download it.

6) Ping / Goodbye

- The client may send `ping` to verify the server replies with `pong`.
- Sending `bye` causes the server to close the session for that connection.

Server behavior and storage

- The server organizes storage per UUID: `server/files/{uuid}/`.
- The server accepts TCP connections on the configured address (default printed as "Listening on :3002").
- Each accepted connection spawns a goroutine that reads opcodes and handles the request stream until the client disconnects or sends `bye`.

Limitations and security notes

- The protocol is unencrypted plain TCP. If you need confidentiality or integrity protection, add TLS (recommended for real-world use).
- There is no authentication beyond possession of a UUID file. UUIDs are not secret keys. Consider adding authentication or access control if needed.
- Filenames are limited to 255 bytes due to uint8 length prefix.

---

## Build and run (quick)

PowerShell quick-run examples:

```powershell
# Start server
cd server
go run .

# Start GUI client (default mode)
cd ..\client
go run .

# Start CLI client
go run . --cli
```

Build examples:

```powershell
cd server; go build -o fsend-server
cd client; go build -tags gio -o fsend-client.exe
```

---

## Developer notes

- If you want to implement a third-party client, I can add a `USAGE.md` that documents the binary protocol byte-for-byte and include example hex dumps and small client snippets in other languages.
- Integration tests that launch server + two clients could be added to validate end-to-end upload/send/download flows.

---

Made with ❤️ by Jayac & Leonwijng
