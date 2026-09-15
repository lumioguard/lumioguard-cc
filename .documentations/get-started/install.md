---
description: Install the lumioguard-cc binary on macOS, Linux or Windows.
---

# Install

lumioguard CC is a single program with nothing else to install. Git is needed only to compare with
earlier versions of your code.

!!! tip "Using a coding agent?"
    Your agent can install it for you. See [Coding agents](../guides/coding-agents.md#install-with-an-agent).

## Download a release

Releases are on the [GitHub releases page](https://github.com/lumiostack/lumioguard-cc/releases). Pick
the file for your computer:

| Computer | File |
| --- | --- |
| Linux, Intel or AMD | `lumioguard-cc_VERSION_linux_amd64.tar.gz` |
| Linux, ARM | `lumioguard-cc_VERSION_linux_arm64.tar.gz` |
| Mac with Apple silicon | `lumioguard-cc_VERSION_darwin_arm64.tar.gz` |
| Mac with Intel | `lumioguard-cc_VERSION_darwin_amd64.tar.gz` |
| Windows, Intel or AMD | `lumioguard-cc_VERSION_windows_amd64.zip` |
| Windows, ARM | `lumioguard-cc_VERSION_windows_arm64.zip` |

Each release also has a `SHA256SUMS` file. Checking the download against it confirms the file arrived
complete and unchanged.

The commands below use version `0.1.0`. Replace it with the latest version on the releases page.

=== "macOS and Linux"

    ```bash
    VERSION=0.1.0
    NAME=lumioguard-cc_${VERSION}_linux_amd64        # pick your file from the table
    BASE=https://github.com/lumiostack/lumioguard-cc/releases/download/v${VERSION}

    curl -fsSLO "$BASE/$NAME.tar.gz"
    curl -fsSLO "$BASE/SHA256SUMS"
    # Must print OK. On macOS, use shasum -a 256 -c - instead of sha256sum -c -
    grep " $NAME.tar.gz\$" SHA256SUMS | sha256sum -c -

    tar -xzf "$NAME.tar.gz"
    mkdir -p ~/.local/bin
    cp "$NAME/lumioguard-cc" "$NAME/THIRD-PARTY-NOTICES.md" ~/.local/bin/
    ```

    If `~/.local/bin` is not on your `PATH`, add `export PATH="$HOME/.local/bin:$PATH"` to your shell
    profile.

=== "Windows"

    Run in PowerShell:

    ```powershell
    $Version = "0.1.0"
    $Name = "lumioguard-cc_${Version}_windows_amd64"
    $Base = "https://github.com/lumiostack/lumioguard-cc/releases/download/v$Version"

    Invoke-WebRequest "$Base/$Name.zip" -OutFile "$Name.zip"
    Invoke-WebRequest "$Base/SHA256SUMS" -OutFile SHA256SUMS
    $Line = (Select-String -Path SHA256SUMS -Pattern "$Name.zip").Line
    $Expected = ($Line -split '\s+')[0]
    (Get-FileHash "$Name.zip" -Algorithm SHA256).Hash -eq $Expected   # must print True

    Expand-Archive "$Name.zip" -DestinationPath .
    $Target = "$env:LOCALAPPDATA\Programs\lumioguard-cc"
    New-Item -ItemType Directory -Force $Target | Out-Null
    Copy-Item "$Name\lumioguard-cc.exe", "$Name\THIRD-PARTY-NOTICES.md" $Target
    ```

    To run it from any folder, add that folder to your user `PATH`, then open a new terminal:

    ```powershell
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$Target", "User")
    ```

=== "With Go"

    If you have Go 1.27 or later:

    ```bash
    go install github.com/lumiostack/lumioguard-cc/cmd/lumioguard-cc@latest
    ```

    The program goes to `$(go env GOPATH)/bin`.

!!! note "Keep the notices file"
    `THIRD-PARTY-NOTICES.md` holds the licence notices of the open source parts inside the program. Keep
    it next to the binary if you copy the program to other machines.

## Check it works

```bash
lumioguard-cc --version
```

It prints the version number, such as `0.1.0`.

## Next step

[:octicons-arrow-right-24: Run your first check](quick-start.md)
