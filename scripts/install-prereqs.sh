#!/usr/bin/env bash
set -euo pipefail

# HomeAPI - Prerequisites Installer
# Installs Go, Node.js, and GCC required to build HomeAPI.
# Supports: Ubuntu/Debian, Fedora/RHEL/CentOS, macOS (Homebrew), Arch Linux

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

GO_MIN_VERSION="1.21"
NODE_MIN_VERSION="18"

# --- Version checking helpers ---

version_ge() {
    # Returns 0 if $1 >= $2 (semantic version comparison)
    printf '%s\n%s\n' "$2" "$1" | sort -V -C
}

check_go() {
    if command -v go &>/dev/null; then
        local ver
        ver=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+(\.[0-9]+)?' || true)
        if [ -n "$ver" ] && version_ge "$ver" "$GO_MIN_VERSION"; then
            info "Go $ver found (>= $GO_MIN_VERSION required)"
            return 0
        fi
    fi
    return 1
}

check_node() {
    if command -v node &>/dev/null; then
        local ver
        ver=$(node --version | sed 's/^v//')
        local major
        major=$(echo "$ver" | cut -d. -f1)
        if [ "$major" -ge "$NODE_MIN_VERSION" ] 2>/dev/null; then
            info "Node.js $ver found (>= $NODE_MIN_VERSION required)"
            return 0
        fi
    fi
    return 1
}

check_npm() {
    if command -v npm &>/dev/null; then
        info "npm $(npm --version) found"
        return 0
    fi
    return 1
}

check_gcc() {
    if command -v gcc &>/dev/null; then
        info "GCC $(gcc -dumpversion) found"
        return 0
    fi
    return 1
}

check_make() {
    if command -v make &>/dev/null; then
        info "make found"
        return 0
    fi
    return 1
}

# --- OS detection ---

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        case "$ID" in
            ubuntu|debian|pop|linuxmint|elementary) echo "debian" ;;
            fedora)                                  echo "fedora" ;;
            centos|rhel|rocky|alma|ol)               echo "rhel" ;;
            arch|manjaro|endeavouros)                 echo "arch" ;;
            *)                                       echo "unknown-linux" ;;
        esac
    elif [ "$(uname)" = "Darwin" ]; then
        echo "macos"
    else
        echo "unknown"
    fi
}

# --- Installers ---

install_debian() {
    info "Detected Debian/Ubuntu-based system"
    sudo apt-get update -qq

    if ! check_gcc || ! check_make; then
        info "Installing build-essential (GCC, make)..."
        sudo apt-get install -y -qq build-essential
    fi

    if ! check_go; then
        info "Installing Go..."
        sudo apt-get install -y -qq golang-go 2>/dev/null || install_go_binary
    fi

    if ! check_node; then
        info "Installing Node.js via NodeSource..."
        if command -v curl &>/dev/null; then
            curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
            sudo apt-get install -y -qq nodejs
        else
            sudo apt-get install -y -qq curl
            curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
            sudo apt-get install -y -qq nodejs
        fi
    fi
}

install_fedora() {
    info "Detected Fedora"
    if ! check_gcc || ! check_make; then
        info "Installing GCC and make..."
        sudo dnf install -y gcc make
    fi
    if ! check_go; then
        info "Installing Go..."
        sudo dnf install -y golang || install_go_binary
    fi
    if ! check_node; then
        info "Installing Node.js..."
        sudo dnf install -y nodejs npm || install_node_binary
    fi
}

install_rhel() {
    info "Detected RHEL/CentOS"
    if ! check_gcc || ! check_make; then
        info "Installing GCC and make..."
        sudo yum install -y gcc make
    fi
    if ! check_go; then
        info "Installing Go..."
        sudo yum install -y golang || install_go_binary
    fi
    if ! check_node; then
        info "Installing Node.js..."
        if command -v curl &>/dev/null; then
            curl -fsSL https://rpm.nodesource.com/setup_20.x | sudo bash -
            sudo yum install -y nodejs
        else
            sudo yum install -y curl
            curl -fsSL https://rpm.nodesource.com/setup_20.x | sudo bash -
            sudo yum install -y nodejs
        fi
    fi
}

install_arch() {
    info "Detected Arch Linux"
    if ! check_gcc || ! check_make; then
        info "Installing base-devel..."
        sudo pacman -S --noconfirm --needed base-devel
    fi
    if ! check_go; then
        info "Installing Go..."
        sudo pacman -S --noconfirm go
    fi
    if ! check_node; then
        info "Installing Node.js..."
        sudo pacman -S --noconfirm nodejs npm
    fi
}

install_macos() {
    info "Detected macOS"

    if ! command -v brew &>/dev/null; then
        error "Homebrew not found. Install it from https://brew.sh and re-run this script."
    fi

    if ! check_gcc; then
        info "Installing Xcode command line tools (provides GCC/clang)..."
        xcode-select --install 2>/dev/null || true
    fi
    if ! check_go; then
        info "Installing Go..."
        brew install go
    fi
    if ! check_node; then
        info "Installing Node.js..."
        brew install node
    fi
}

# --- Fallback binary installers ---

install_go_binary() {
    local arch
    case "$(uname -m)" in
        x86_64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local version="1.22.5"
    local url="https://go.dev/dl/go${version}.${os}-${arch}.tar.gz"

    info "Downloading Go $version from $url ..."
    curl -fsSL "$url" -o /tmp/go.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz

    if ! echo "$PATH" | grep -q "/usr/local/go/bin"; then
        echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh >/dev/null
        export PATH=$PATH:/usr/local/go/bin
    fi
    info "Go installed to /usr/local/go"
}

install_node_binary() {
    local arch
    case "$(uname -m)" in
        x86_64)  arch="x64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
    local version="20.18.0"
    local url="https://nodejs.org/dist/v${version}/node-v${version}-linux-${arch}.tar.xz"

    info "Downloading Node.js $version from $url ..."
    curl -fsSL "$url" -o /tmp/node.tar.xz
    sudo tar -C /usr/local --strip-components=1 -xJf /tmp/node.tar.xz
    rm -f /tmp/node.tar.xz
    info "Node.js installed to /usr/local"
}

# --- Main ---

main() {
    echo ""
    echo "================================="
    echo "  HomeAPI Prerequisites Installer"
    echo "================================="
    echo ""

    local os
    os=$(detect_os)

    case "$os" in
        debian)        install_debian ;;
        fedora)        install_fedora ;;
        rhel)          install_rhel ;;
        arch)          install_arch ;;
        macos)         install_macos ;;
        unknown-linux) warn "Unrecognized Linux distro. Attempting fallback installs..."
                       check_gcc  || error "GCC not found. Please install it manually."
                       check_make || error "make not found. Please install it manually."
                       check_go   || install_go_binary
                       check_node || install_node_binary
                       ;;
        *)             error "Unsupported OS: $(uname). Please install Go, Node.js, and GCC manually." ;;
    esac

    echo ""
    echo "================================="
    echo "  Verifying installations"
    echo "================================="
    echo ""

    local ok=true
    check_go    || { warn "Go not found or version too old (need >= $GO_MIN_VERSION)"; ok=false; }
    check_node  || { warn "Node.js not found or version too old (need >= $NODE_MIN_VERSION)"; ok=false; }
    check_npm   || { warn "npm not found"; ok=false; }
    check_gcc   || { warn "GCC not found"; ok=false; }
    check_make  || { warn "make not found"; ok=false; }

    echo ""
    if [ "$ok" = true ]; then
        info "All prerequisites installed. You can now run: make build"
    else
        warn "Some prerequisites are missing. Check the warnings above."
    fi
}

main "$@"
