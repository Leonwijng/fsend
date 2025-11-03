# fsend - Cross-Platform File Sharing

Fast and simple file sharing application with GUI support.

## Features

- 🎨 Native GUI with Material Design
- 📂 File picker (Windows native, terminal input on Linux/macOS)
- 📋 Copy UUID to clipboard
- ⬇️ Upload, download, and send files
- 💻 CLI mode available
- 🔒 UUID-based file storage

## Platform Support

✅ **Windows** - Full native support with GUI file picker
✅ **Linux** - Full support (file paths via terminal input)
✅ **macOS** - Full support (file paths via terminal input)

## Building

### Windows

```powershell
cd client
go build -tags gio -ldflags="-H windowsgui -s -w" -o fsend.exe
```

### Linux

```bash
cd client
go build -tags gio -o fsend
```

### macOS

```bash
cd client
go build -tags gio -o fsend
```

## Running

### GUI Mode (Default)
Simply double-click the executable or run:
```bash
./fsend
```

### CLI Mode
```bash
./fsend --cli
```

## Dependencies

- **Gio** - Pure Go GUI framework (cross-platform)
- **clipboard** - Cross-platform clipboard support
- **dialog** (Windows only) - Native file picker on Windows

All dependencies are automatically managed by Go modules.

## Server

To run the server:
```bash
cd server
go run .
```

The server listens on `localhost:3002` by default.

## Architecture

- **client/** - Main GUI client with icon
- **client2/** - Secondary client instance (identical features)
- **server/** - File storage server

## Notes

- On Windows: Native file picker dialog
- On Linux/macOS: File paths are entered in the terminal when prompted
- All platforms support CLI mode with interactive menu
- Icon is embedded on Windows only (using winres)

## License

See LICENSE file for details.
