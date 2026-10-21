# AIDA64 push client

A small Go program for Windows and Linux that reads the PC's RAM, CPU and disk and pushes them to the device's [AIDA64 face](../README.md) with a face push token (device decision D35). It came from the device repository's `aida64-client-go`.

## What it reads
- RAM usage, free and load, CPU name, clock and load, and disk usage, through `gopsutil`.
- Linux temperatures and the CPU fan from `/sys/class/hwmon`.
- Windows temperatures and the CPU fan from LibreHardwareMonitor's JSON (URL or file).
- RAM speed, best effort: `dmidecode` on Linux, `wmic` or PowerShell on Windows.
- The disk's name from the mounted partition.

## Set up
1. Install `example.aida64` on the device's SD card and add **AIDA64** to the face cycle on the Device Console (**Display**).
2. On the same page, under **Push tokens**, create a token for AIDA64 and copy it. It is shown once.
3. Run the client with it.

## Build

```bash
cd plugins/aida64/client
go build -o aida64-push-client ./cmd/aida64-push-client
GOOS=windows GOARCH=amd64 go build -o dist/aida64-push-client-windows-amd64.exe ./cmd/aida64-push-client
```

## Run

```bash
./aida64-push-client --device http://192.168.1.50 --token aitbase_face_... --interval 1s
```

The token can come from `AIDA64_FACE_TOKEN` instead of `--token`. The client posts `{"values": {"face.1": ..., "face.16": ...}}` to `POST /api/v1/plugins/face-values` with `Authorization: Bearer <token>`. The keys are in the [face's README](../README.md#the-faces-values). Values are cut to 64 bytes, and control and bidi characters are dropped.

The device allows 2 writes a second per token (bursts of 5), so keep `--interval` at 500ms or more.

### Windows: LibreHardwareMonitor
Enable LibreHardwareMonitor's web server (default `http://127.0.0.1:8085/data.json`), or point at a saved file:

```bash
./aida64-push-client --token ... --lhm-url http://127.0.0.1:8085/data.json
./aida64-push-client --token ... --lhm-file C:\\temp\\lhm.json
```

## Options
- `--device`: the device's base URL.
- `--token`: the face push token (default `$AIDA64_FACE_TOKEN`).
- `--interval`, `--timeout`: push interval and HTTP timeout.
- `--storage-usage`, `--storage-path`, `--storage-disk`, `--storage-temp`: disk overrides. With `--storage-usage`, the disk bar (`face.13`) and total (`face.15`) come from that text.
- `--cpu-name`, `--ram-speed` (`--ram-type` is a deprecated alias): overrides.
- `--once`: send once and exit.
- `--lhm-url`, `--lhm-file`: LibreHardwareMonitor on Windows.

The old `--force-aida-mode` and `--persist-mode` are gone: they needed the owner's session, which the client never had. Choose the face on the Device Console instead.

## Notes
- Linux RAM speed needs `dmidecode`, usually as root. Without it, pass `--ram-speed`.
