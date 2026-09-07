"""Artifact guard tests; fixtures do not establish native build acceptance."""
import importlib.util
from pathlib import Path
import plistlib
import struct
import tempfile
import unittest
import zipfile

spec = importlib.util.spec_from_file_location("verify_ipa", Path(__file__).with_name("verify-ios-ipa.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class VerifyIPATest(unittest.TestCase):
    def check(self, *, platform=2, bundle_id="ai.chimii.mobile", extra=b"", signed=False, embedded=True, url="https://chimii.com"):
        with tempfile.TemporaryDirectory() as directory:
            ipa = Path(directory) / "fixture.ipa"
            info = {
                "CFBundleIdentifier": bundle_id, "CFBundleShortVersionString": "0.2.1",
                "CFBundleVersion": "4", "CFBundleExecutable": "Chimii",
                "CFBundleSupportedPlatforms": ["iPhoneOS"], "MinimumOSVersion": "15.1",
            }
            # A simulator ARM64 binary is deliberately paired with device metadata
            # in the negative test: checking Info.plist alone is insufficient.
            binary = struct.pack("<8I6I", 0xFEEDFACF, 0x0100000C, 0, 2, 1, 24, 0, 0, 0x32, 24, platform, 0, 0, 0)
            with zipfile.ZipFile(ipa, "w") as archive:
                archive.writestr("Payload/Chimii.app/Info.plist", plistlib.dumps(info))
                archive.writestr("Payload/Chimii.app/Chimii", binary)
                if embedded:
                    archive.writestr("Payload/Chimii.app/main.jsbundle", bytes.fromhex("c61fbc03c103191f") + url.encode() + extra)
                if signed:
                    archive.writestr("Payload/Chimii.app/embedded.mobileprovision", b"profile")
            return module.verify(ipa, "production", "https://chimii.com", "https://chimii.com", "0.2.1", "4")

    def test_accepts_unsigned_device_with_production_bundle(self):
        result = self.check()
        self.assertEqual(result["platform"], "iPhoneOS")
        self.assertEqual(len(result["sha256"]), 64)

    def test_rejects_simulator_binary_despite_device_metadata(self):
        with self.assertRaisesRegex(ValueError, "not built for iOS devices"):
            self.check(platform=7)

    def test_rejects_wrong_environment(self):
        with self.assertRaisesRegex(ValueError, "bundle identifier"):
            self.check(bundle_id="ai.chimii.mobile.staging")
        with self.assertRaisesRegex(ValueError, "service URL"):
            self.check(url="https://staging.chimii.com")

    def test_rejects_fixture_credentials(self):
        with self.assertRaisesRegex(ValueError, "Fixture configuration"):
            self.check(extra=b"local-fixture-token")

    def test_rejects_signed_or_missing_bundle(self):
        with self.assertRaisesRegex(ValueError, "unsigned"):
            self.check(signed=True)
        with self.assertRaises(KeyError):
            self.check(embedded=False)


if __name__ == "__main__":
    unittest.main()
