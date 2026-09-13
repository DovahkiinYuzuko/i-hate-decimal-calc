#!/usr/bin/env bash
set -e

LOCAL_MODE=false
for arg in "$@"; do
    if [ "$arg" = "--local" ] || [ "$arg" = "-l" ]; then
        LOCAL_MODE=true
    fi
done

REPO="DovahkiinYuzuko/i-hate-decimal-calc"
IHD_HOME="$HOME/.ihd"
INSTALL_DIR="$IHD_HOME/bin"
EXAM_DIR="$IHD_HOME/exam"

mkdir -p "$INSTALL_DIR"

if [ "$LOCAL_MODE" = true ]; then
    echo "Installing ihd from local repository..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    (cd "$SCRIPT_DIR" && go build -ldflags="-s -w" -o "$INSTALL_DIR/ihd" ./cmd/ihd)
    chmod +x "$INSTALL_DIR/ihd"
    if [ -d "$SCRIPT_DIR/exam" ]; then
        cp -r "$SCRIPT_DIR/exam" "$IHD_HOME/"
        chmod +x "$EXAM_DIR/exam.sh" 2>/dev/null || true
    fi
else
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux)
            if [ "$ARCH" = "x86_64" ]; then
                ASSET_NAME="ihd-linux-amd64.tar.gz"
            elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
                ASSET_NAME="ihd-linux-arm64.tar.gz"
            else
                echo "Error: Unsupported Linux architecture: $ARCH"
                exit 1
            fi
            ;;
        Darwin)
            if [ "$ARCH" = "x86_64" ]; then
                ASSET_NAME="ihd-darwin-amd64.tar.gz"
            elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
                ASSET_NAME="ihd-darwin-arm64.tar.gz"
            else
                echo "Error: Unsupported macOS architecture: $ARCH"
                exit 1
            fi
            ;;
        *)
            echo "Error: Unsupported operating system: $OS"
            echo "For Windows, please use install.ps1 via PowerShell."
            exit 1
            ;;
    esac

    echo "Fetching latest release for $REPO..."
    RELEASE_URL="https://api.github.com/repos/$REPO/releases/latest"
    if command -v curl >/dev/null 2>&1; then
        RELEASE_JSON="$(curl -fsSL "$RELEASE_URL")"
    elif command -v wget >/dev/null 2>&1; then
        RELEASE_JSON="$(wget -qO- "$RELEASE_URL")"
    else
        echo "Error: curl or wget is required to install ihd."
        exit 1
    fi

    TAG="$(echo "$RELEASE_JSON" | grep '"tag_name":' | head -n1 | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')"
    if [ -z "$TAG" ]; then
        echo "Error: Failed to determine latest release tag from GitHub API."
        exit 1
    fi

    echo "Latest release: $TAG"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG/$ASSET_NAME"
    TMP_DIR="$(mktemp -d)"
    trap 'rm -rf "$TMP_DIR"' EXIT

    echo "Downloading $DOWNLOAD_URL..."
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET_NAME"
    else
        wget -qO "$TMP_DIR/$ASSET_NAME" "$DOWNLOAD_URL"
    fi

    tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$TMP_DIR"

    cp "$TMP_DIR/ihd" "$INSTALL_DIR/ihd"
    chmod +x "$INSTALL_DIR/ihd"

    if [ -d "$TMP_DIR/exam" ]; then
        cp -r "$TMP_DIR/exam" "$IHD_HOME/"
        chmod +x "$EXAM_DIR/exam.sh" 2>/dev/null || true
    fi
fi

echo ""
echo "Successfully installed ihd to $INSTALL_DIR/ihd"
if [ -d "$EXAM_DIR" ]; then
    echo "Mathematical Showcase & Cookbook installed to: $EXAM_DIR"
    echo "Run the live showcase: $EXAM_DIR/exam.sh"
fi
echo ""

# Check PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo "Note: $INSTALL_DIR is not in your PATH."
        echo "Add the following line to your shell configuration file (~/.bashrc, ~/.zshrc, or ~/.profile):"
        echo ""
        echo "  export PATH=\"\$HOME/.ihd/bin:\$PATH\""
        echo ""
        echo "Then reload your shell or run: source ~/.zshrc (or ~/.bashrc)"
        ;;
esac
