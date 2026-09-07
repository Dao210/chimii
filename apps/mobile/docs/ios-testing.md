# iOS personal testing

The `*-unsigned.ipa` is a standalone Release build for **physical iPhones and
iPads running iOS 15.1 or later**. It embeds Hermes and the application bundle;
Metro and a local development server are not needed. Production builds connect
to `https://chimii.com`.

This package requires personal signing before installation. It is not a
TestFlight or App Store distribution. CI compilation and package checks do not
establish successful installation or runtime acceptance on your device.

## Install on your iPhone

1. Download the `*-unsigned.ipa` from this test release. The `.sha256` file is its
   checksum; `.checks.json` records device architecture, version and bundle checks.
   A failed build contains diagnostics only and has no installable package.
2. Install [Sideloadly](https://sideloadly.io/) on your Mac or Windows computer.
3. Connect and trust your iPhone by USB. Select it in Sideloadly, drag in the IPA,
   and use your own Apple ID to sign and install it. Enter credentials only in
   the signing application, never in GitHub secrets or this repository.
4. Follow the device's prompts to trust the developer profile and, where
   required, enable Developer Mode. Open Chimii and sign into your account.

A free Apple ID can sign personal test apps, normally for **seven days**. Re-sign
before expiry to keep testing. Use the same Apple ID and signing bundle ID for
updates. See the [official Sideloadly FAQ](https://sideloadly.io/faq) for current
device and account limits.

Check login, Build → Circuit → Block → Creations navigation, 3D loading, saving
and reopening a creation, and background/foreground recovery on the device.

## Build again

The **iOS Test IPA** GitHub workflow supports manual dispatch (production or
staging). The output is retained as an Actions artifact for 14 days. A dedicated
`ios-test-*` tag builds production and publishes a downloadable **prerelease**;
it does not become the latest stable release or trigger the main `v*.*.*` release.

Push the reviewed commit first, then use a fresh tag for each test delivery:

```sh
git tag ios-test-v0.2.1-1
git push origin ios-test-v0.2.1-1
```

Git uses the repository's configured remote and SSH identity. The workflow uses
GitHub's short-lived job token to publish; no personal access token, Apple
Developer membership, signing certificate or provisioning profile is needed.
Build numbers come from the workflow run number. Failed builds publish compressed
build logs for diagnosis instead of an IPA.

Local equivalent, from `apps/mobile/`, with Xcode 26.3, CocoaPods, Node 22 and pnpm:

```sh
APP_ENV=production IOS_BUILD_NUMBER=1 pnpm ios:unsigned
```

The script loads the selected committed `.env` file, generates native config via
`scripts/ios-run.sh --prebuild-only`, archives an ARM64 iPhoneOS app with signing
disabled, and checks the actual IPA before placing it in `artifacts/ios/`.
