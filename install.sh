#!/bin/bash
set -euo pipefail

REPO="DeraDream/sublinkX"
BRANCH="main"
INSTALL_DIR="/usr/local/bin/sublink"
SERVICE_FILE="/etc/systemd/system/sublink.service"
GO_VERSION="1.22.12"
NODE_VERSION="20.18.3"

if [ "$(id -u)" != "0" ]; then
    echo "该脚本必须以 root 身份运行。"
    exit 1
fi

install_build_dependencies() {
    if command -v curl >/dev/null 2>&1 &&
       command -v tar >/dev/null 2>&1 &&
       command -v xz >/dev/null 2>&1; then
        return
    fi

    echo "正在安装构建依赖..."
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update
        DEBIAN_FRONTEND=noninteractive apt-get install -y curl ca-certificates tar xz-utils
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y curl ca-certificates tar xz
    elif command -v yum >/dev/null 2>&1; then
        yum install -y curl ca-certificates tar xz
    elif command -v apk >/dev/null 2>&1; then
        apk add --no-cache curl ca-certificates tar xz
    else
        echo "无法识别系统包管理器，请先安装 curl、tar 和 xz。"
        exit 1
    fi
}

case "$(uname -m)" in
    x86_64)
        BUILD_ARCH="amd64"
        NODE_ARCH="x64"
        ;;
    aarch64|arm64)
        BUILD_ARCH="arm64"
        NODE_ARCH="arm64"
        ;;
    *)
        echo "不支持的机器类型: $(uname -m)"
        exit 1
        ;;
esac

install_build_dependencies

BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT

echo "正在从 $REPO 的 $BRANCH 分支下载源码..."
curl --fail --location --retry 3 \
    "https://github.com/$REPO/archive/refs/heads/$BRANCH.tar.gz" \
    -o "$BUILD_DIR/source.tar.gz"
tar -xzf "$BUILD_DIR/source.tar.gz" -C "$BUILD_DIR"
SOURCE_DIR=$(find "$BUILD_DIR" -mindepth 1 -maxdepth 1 -type d -name 'sublinkX-*' | head -n 1)

if [ -z "$SOURCE_DIR" ]; then
    echo "源码解压失败。"
    exit 1
fi

echo "正在准备 Go $GO_VERSION 和 Node.js $NODE_VERSION..."
curl --fail --location --retry 3 \
    "https://go.dev/dl/go${GO_VERSION}.linux-${BUILD_ARCH}.tar.gz" \
    -o "$BUILD_DIR/go.tar.gz"
tar -xzf "$BUILD_DIR/go.tar.gz" -C "$BUILD_DIR"

curl --fail --location --retry 3 \
    "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${NODE_ARCH}.tar.xz" \
    -o "$BUILD_DIR/node.tar.xz"
tar -xJf "$BUILD_DIR/node.tar.xz" -C "$BUILD_DIR"

export PATH="$BUILD_DIR/go/bin:$BUILD_DIR/node-v${NODE_VERSION}-linux-${NODE_ARCH}/bin:$PATH"
export CGO_ENABLED=0

echo "正在构建前端..."
cd "$SOURCE_DIR/webs"
corepack enable
corepack prepare pnpm@8.15.6 --activate
HUSKY=0 pnpm install --no-frozen-lockfile
pnpm exec vite build --mode production

rm -rf "$SOURCE_DIR/static"
cp -R "$SOURCE_DIR/webs/dist" "$SOURCE_DIR/static"

echo "正在构建后端..."
cd "$SOURCE_DIR"
go build -trimpath -ldflags="-s -w" -o "$BUILD_DIR/sublink" .

mkdir -p "$INSTALL_DIR/db" "$INSTALL_DIR/template" "$INSTALL_DIR/logs"
if systemctl is-active --quiet sublink 2>/dev/null; then
    systemctl stop sublink
fi
install -m 755 "$BUILD_DIR/sublink" "$INSTALL_DIR/sublink"

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Sublink Service
After=network.target

[Service]
ExecStart=$INSTALL_DIR/sublink
WorkingDirectory=$INSTALL_DIR
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

curl --fail --location \
    -H "Cache-Control: no-cache" \
    -H "Pragma: no-cache" \
    "https://raw.githubusercontent.com/$REPO/$BRANCH/menu.sh" \
    -o /usr/bin/sublink
chmod 755 /usr/bin/sublink

systemctl daemon-reload
systemctl enable --now sublink

echo "安装完成，当前版本: $("$INSTALL_DIR/sublink" --version)"
echo "默认账号 admin，默认密码 123456，默认端口 8000"
echo "输入 sublink 可呼出管理菜单"
