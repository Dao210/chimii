#!/usr/bin/env python3
"""Validate the final device IPA, including its bundled service environment."""
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import plistlib
import struct
import sys
import zipfile


def require(condition, message):
    if not condition:
        raise ValueError(message)


def verify(ipa, environment, api_url, web_url, version, build_number):
    expected_id = "ai.chimii.mobile" + (".staging" if environment == "staging" else "")
    require(environment in ("production", "staging"), "Invalid environment")
    with zipfile.ZipFile(ipa) as archive:
        names = archive.namelist()
        require(all(not n.startswith("/") and ".." not in PurePosixPath(n).parts for n in names), "Unsafe IPA path")
        roots = [n for n in names if n.startswith("Payload/") and n.endswith(".app/Info.plist") and n.count("/") == 2]
        require(len(roots) == 1, "Expected one Payload application")
        root = roots[0].removesuffix("Info.plist")
        info = plistlib.loads(archive.read(roots[0]))
        require(info.get("CFBundleIdentifier") == expected_id, "Unexpected bundle identifier")
        require(info.get("CFBundleShortVersionString") == version, "Version mismatch")
        require(info.get("CFBundleVersion") == str(build_number), "Build number mismatch")
        require(info.get("CFBundleSupportedPlatforms") == ["iPhoneOS"], "Not an iPhone device build")
        require(root + "embedded.mobileprovision" not in names, "Expected an unsigned package")
        require(root + "_CodeSignature/CodeResources" not in names, "App already contains a signature")
        executable = info.get("CFBundleExecutable", "")
        require(executable and "/" not in executable, "Invalid executable name")
        binary = archive.read(root + executable)
        require(len(binary) >= 32, "Missing Mach-O executable")
        magic, cpu, _, file_type, ncmds, sizeofcmds, _, _ = struct.unpack_from("<8I", binary)
        require(magic == 0xFEEDFACF and cpu == 0x0100000C and file_type == 2, "Expected an ARM64 Mach-O executable")
        require(32 + sizeofcmds <= len(binary), "Truncated Mach-O commands")
        offset, platform = 32, None
        for _ in range(ncmds):
            require(offset + 8 <= 32 + sizeofcmds, "Invalid Mach-O command")
            cmd, size = struct.unpack_from("<2I", binary, offset)
            require(size >= 8 and offset + size <= 32 + sizeofcmds, "Invalid command size")
            if cmd == 0x32:
                require(size >= 24, "Truncated build version command")
                platform = struct.unpack_from("<I", binary, offset + 8)[0]
            if cmd == 0x2C:
                require(size >= 24 and struct.unpack_from("<I", binary, offset + 16)[0] == 0, "Encrypted executable")
            offset += size
        require(offset == 32 + sizeofcmds, "Invalid Mach-O command count")
        require(platform == 2, "Executable is not built for iOS devices")
        bundle = archive.read(root + "main.jsbundle")
        require(bundle[:8].hex() == "c61fbc03c103191f", "Missing embedded Hermes bytecode")
        for url in (api_url, web_url):
            require(url and url.encode() in bundle, "Missing configured service URL")
            if environment == "production":
                require(url.startswith("https://"), "Production requires HTTPS")
        if environment == "production":
            require(not info.get("NSAppTransportSecurity", {}).get("NSAllowsArbitraryLoads", False), "Production permits arbitrary cleartext")
        require(all(v not in bundle for v in (b"http://10.0.2.2:", b"local-fixture-token")), "Fixture configuration found")
        with open(ipa, "rb") as source:
            digest = hashlib.file_digest(source, "sha256").hexdigest()
        return {
            "version": version, "build_number": str(build_number),
            "bundle_identifier": expected_id, "platform": "iPhoneOS", "architecture": "arm64",
            "minimum_ios": info.get("MinimumOSVersion"), "unsigned": True,
            "hermes_embedded": True, "api_url": api_url, "web_url": web_url,
            "fixture_configuration_absent": True, "bytes": Path(ipa).stat().st_size,
            "sha256": digest,
        }


if __name__ == "__main__":
    path = Path(sys.argv[1])
    package = Path(__file__).resolve().parent.parent / "package.json"
    result = verify(path, os.environ["APP_ENV"], os.environ["EXPO_PUBLIC_API_URL"], os.environ["EXPO_PUBLIC_WEB_URL"], json.loads(package.read_text())["version"], os.environ["IOS_BUILD_NUMBER"])
    path.with_suffix(".checks.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result, indent=2))
