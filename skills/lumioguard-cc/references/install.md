# Installing the lumioguard-cc CLI

It is a single binary with no runtime dependencies. Use the download URL or version the user gave;
otherwise use the latest release from https://github.com/lumiostack/lumioguard-cc/releases.

1. **Find the version.** https://github.com/lumiostack/lumioguard-cc/releases/latest redirects to a
   URL ending in `/tag/vX.Y.Z`.

2. **Pick the file.** Check the platform with `uname -sm`, or `$env:PROCESSOR_ARCHITECTURE` on
   Windows.

   | Platform | File |
   | --- | --- |
   | Linux x86-64 / ARM64 | `lumioguard-cc_X.Y.Z_linux_amd64.tar.gz` / `_linux_arm64.tar.gz` |
   | macOS Intel / Apple silicon | `lumioguard-cc_X.Y.Z_darwin_amd64.tar.gz` / `_darwin_arm64.tar.gz` |
   | Windows x86-64 / ARM64 | `lumioguard-cc_X.Y.Z_windows_amd64.zip` / `_windows_arm64.zip` |

   The archive and `SHA256SUMS` are both at
   `https://github.com/lumiostack/lumioguard-cc/releases/download/vX.Y.Z/<file>`.

3. **Download and verify.** Download both into a temp folder. Compare the archive's SHA-256 with its
   line in `SHA256SUMS`. Use `sha256sum` on Linux, `shasum -a 256` on macOS, and `Get-FileHash` or
   `certutil -hashfile <file> SHA256` on Windows. **If the hashes differ, delete the download and
   stop.**

4. **Install without administrator rights.** The archive holds a folder with the binary, `README.md`
   and `THIRD-PARTY-NOTICES.md`.
   - **macOS and Linux:** copy the binary to `~/.local/bin` and run `chmod +x` on it.
   - **Windows:** copy it to `%LOCALAPPDATA%\Programs\lumioguard-cc\`.

   Keep `THIRD-PARTY-NOTICES.md` next to the binary, because the licences require it. Delete the
   temp folder.

5. **Confirm:** run the binary with `--version`. If its folder is not on `PATH`, tell the user, and
   ask before adding it.

If downloads are blocked and Go 1.27 or later is installed, run
`go install github.com/lumiostack/lumioguard-cc/cmd/lumioguard-cc@latest`. If that fails too, stop and
tell the user.
