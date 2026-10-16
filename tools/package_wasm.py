#!/usr/bin/env python3
"""Packages a plugin built with wasi-sdk for the SD card of a development device.

Writes <out>/<id>-<version>-wasm.zip, which unzips at the root of the SD card into the
directory the device's plugin supervisor runs community plugins from:

    aitbase/plugins/<id>/<entry>.wasm
    aitbase/plugins/<id>/<screen>.ui.json   (one per ui.screens entry)
    aitbase/plugins/<id>/plugin.json

Before packaging, it checks the module against the device runner's host ABI
(aitbase-vision-plugin-sdk docs/wasm-abi.md): only the imports the device provides, the
required exports, no start section and the size limit. A module that fails here would be
refused on the device. The zip is deterministic: sorted entries, fixed timestamps and modes, no compression.

Interim format: the planned package is a signed .avp that the device verifies in memory
(docs/package-format.md in the SDK); the device can't install those yet.

Entries are stored, not compressed: deflate output differs between zlib builds, and the
package's SHA-256 must be the same on every machine.

Usage: package_wasm.py <plugin dir> <wasm build dir> <out dir> [--strip <llvm-strip>]
"""
import argparse
import hashlib
import json
import subprocess
import sys
import tempfile
import zipfile
from pathlib import Path

MAX_MODULE_BYTES = 4 << 20
MAX_SCREEN_BYTES = 16384
MAX_SCREENS = 16
ALLOWED_IMPORTS = {
    ("aitbase", "send"), ("aitbase", "log"), ("aitbase", "timer_set"), ("aitbase", "package_read"),
    ("wasi_snapshot_preview1", "fd_write"), ("wasi_snapshot_preview1", "fd_close"),
    ("wasi_snapshot_preview1", "fd_seek"), ("wasi_snapshot_preview1", "clock_time_get"),
    ("wasi_snapshot_preview1", "proc_exit"),
}
REQUIRED_EXPORTS = {
    "memory", "aitbase_abi_version", "aitbase_rx_buffer", "aitbase_start", "aitbase_message",
    "aitbase_timer", "aitbase_stop",
}
FIXED_TIME = (1980, 1, 1, 0, 0, 0)


class PackageError(Exception):
    pass


def _uleb(data, pos):
    result = shift = 0
    while True:
        if pos >= len(data):
            raise PackageError("truncated module")
        byte = data[pos]
        pos += 1
        result |= (byte & 0x7F) << shift
        shift += 7
        if not byte & 0x80:
            return result, pos


def _name(data, pos):
    length, pos = _uleb(data, pos)
    return data[pos:pos + length].decode("utf-8"), pos + length


def check_module(data):
    """Returns (imports, exports); raises PackageError for what the device would refuse."""
    if data[:8] != b"\0asm\x01\0\0\0":
        raise PackageError("not a WebAssembly module (expected wasm32 binary version 1)")
    if len(data) > MAX_MODULE_BYTES:
        raise PackageError(f"module is {len(data)} bytes; the device accepts at most {MAX_MODULE_BYTES}")
    imports, exports = [], set()
    pos = 8
    while pos < len(data):
        section = data[pos]
        size, pos = _uleb(data, pos + 1)
        body, end = pos, pos + size
        if end > len(data):
            raise PackageError("truncated module")
        if section == 2:
            count, body = _uleb(data, body)
            for _ in range(count):
                module, body = _name(data, body)
                field, body = _name(data, body)
                kind = data[body]
                body += 1
                if kind != 0:
                    raise PackageError(f"import {module}.{field} is not a function; the device provides none")
                _, body = _uleb(data, body)
                imports.append((module, field))
        elif section == 7:
            count, body = _uleb(data, body)
            for _ in range(count):
                name, body = _name(data, body)
                body += 1
                _, body = _uleb(data, body)
                exports.add(name)
        elif section == 8:
            raise PackageError("module has a start section; build it as a reactor (the SDK's CMake does)")
        pos = end
    refused = sorted(f"{m}.{f}" for m, f in imports if (m, f) not in ALLOWED_IMPORTS)
    if refused:
        raise PackageError("the device doesn't provide these imports: " + ", ".join(refused) +
                           ". Use aitbase_plugin.h; files, sockets, environment and randomness aren't available")
    missing = sorted(REQUIRED_EXPORTS - exports)
    if missing:
        raise PackageError("missing exports: " + ", ".join(missing) + ". Link aitbase::plugin built with wasi-sdk")
    return imports, exports


def build_package(plugin_dir, build_dir, out_dir, strip):
    manifest_path = plugin_dir / "plugin.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    plugin_id, version, entry = manifest["id"], manifest["version"], manifest["entry"]
    screens = manifest.get("ui", {}).get("screens", [])
    if len(screens) > MAX_SCREENS:
        raise PackageError(f"{len(screens)} screens; the device accepts at most {MAX_SCREENS}")

    module_path = build_dir / entry
    if not module_path.is_file():
        raise PackageError(f"no module at {module_path}; build with the wasi-sdk toolchain first")
    with tempfile.TemporaryDirectory() as tmp:
        stripped = Path(tmp) / f"{entry}.wasm"
        if strip:
            subprocess.run([strip, "-o", str(stripped), str(module_path)], check=True)
        else:
            stripped.write_bytes(module_path.read_bytes())
        module = stripped.read_bytes()
    check_module(module)

    files = {f"{entry}.wasm": module, "plugin.json": manifest_path.read_bytes()}
    for screen in screens:
        data = (build_dir / f"{screen}.ui.json").read_bytes()
        if len(data) > MAX_SCREEN_BYTES:
            raise PackageError(f"{screen}.ui.json is {len(data)} bytes; the device accepts at most {MAX_SCREEN_BYTES}")
        files[f"{screen}.ui.json"] = data

    out_dir.mkdir(parents=True, exist_ok=True)
    package = out_dir / f"{plugin_id}-{version}-wasm.zip"
    prefix = f"aitbase/plugins/{plugin_id}/"
    with zipfile.ZipFile(package, "w", zipfile.ZIP_STORED) as archive:
        for directory in ("aitbase/", "aitbase/plugins/", prefix):
            info = zipfile.ZipInfo(directory, FIXED_TIME)
            info.external_attr = (0o40755 << 16) | 0x10
            archive.writestr(info, b"")
        for name in sorted(files):
            info = zipfile.ZipInfo(prefix + name, FIXED_TIME)
            info.external_attr = 0o100644 << 16
            archive.writestr(info, files[name])
    return package, len(module)


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("plugin_dir", type=Path)
    parser.add_argument("build_dir", type=Path)
    parser.add_argument("out_dir", type=Path)
    parser.add_argument("--strip", help="llvm-strip from wasi-sdk, to drop debug sections")
    args = parser.parse_args()
    try:
        package, module_bytes = build_package(args.plugin_dir, args.build_dir, args.out_dir, args.strip)
    except (PackageError, OSError, KeyError, ValueError, subprocess.CalledProcessError) as exc:
        print(f"package_wasm: {args.plugin_dir}: {exc}", file=sys.stderr)
        return 1
    digest = hashlib.sha256(package.read_bytes()).hexdigest()
    print(f"{package} module={module_bytes} bytes sha256={digest}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
