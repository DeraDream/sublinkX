#!/bin/bash

REPO="DeraDream/sublinkX"
INSTALL_DIR="/usr/local/bin/sublink"
BIN_PATH="$INSTALL_DIR/sublink"
MENU_PATH="/usr/bin/sublink"
SERVICE_FILE="/etc/systemd/system/sublink.service"

function log_step {
    echo
    echo "==> $1"
}

function detect_arch {
    case "$(uname -m)" in
        x86_64)
            echo "sublink_amd64"
            ;;
        aarch64|arm64)
            echo "sublink_arm64"
            ;;
        *)
            echo "不支持的机器类型: $(uname -m)" >&2
            return 1
            ;;
    esac
}

function latest_release {
    curl --fail --show-error --location --retry 3 \
        "https://api.github.com/repos/$REPO/releases/latest" |
        grep '"tag_name":' |
        sed -E 's/.*"([^"]+)".*/\1/'
}

function is_installed {
    [ -x "$BIN_PATH" ] && [ -f "$SERVICE_FILE" ]
}

function is_running {
    systemctl is-active --quiet sublink 2>/dev/null
}

function current_version {
    if [ -x "$BIN_PATH" ]; then
        "$BIN_PATH" --version 2>/dev/null || echo "unknown"
    else
        echo "未安装"
    fi
}

function install_sublink {
    log_step "开始安装 SublinkX"
    curl --fail --show-error --location --retry 3 \
        -H "Cache-Control: no-cache" \
        -H "Pragma: no-cache" \
        "https://raw.githubusercontent.com/$REPO/main/install.sh" |
        bash
}

function uninstall_sublink {
    log_step "停止并卸载 SublinkX"
    if is_running; then
        systemctl stop sublink
    fi
    if systemctl is-enabled --quiet sublink 2>/dev/null; then
        systemctl disable sublink
    fi
    if [ -f "$SERVICE_FILE" ]; then
        rm -f "$SERVICE_FILE"
    fi
    systemctl daemon-reload

    rm -f "$BIN_PATH"
    rm -f "$MENU_PATH"

    read -p "是否删除模板文件、数据库和日志？(y/n): " is_delete
    if [ "$is_delete" = "y" ]; then
        rm -rf "$INSTALL_DIR/db" "$INSTALL_DIR/template" "$INSTALL_DIR/logs"
    fi

    echo "卸载完成"
}

function update_sublink {
    if ! is_installed; then
        echo "当前未安装 SublinkX，请先安装。"
        return 1
    fi

    log_step "获取最新版本信息"
    latest="$(latest_release)"
    if [ -z "$latest" ]; then
        echo "未找到可更新的发行版。"
        return 1
    fi

    version="$(current_version)"
    echo "当前版本: $version"
    echo "最新版本: $latest"
    if [ "$version" = "$latest" ]; then
        echo "当前已经是最新版本。"
        return 0
    fi

    file_name="$(detect_arch)" || return 1
    tmp_bin="$(mktemp)"
    tmp_menu="$(mktemp)"
    trap 'rm -f "$tmp_bin" "$tmp_menu"' RETURN

    log_step "下载主程序: $file_name"
    curl --fail --show-error --location --retry 3 \
        "https://github.com/$REPO/releases/download/$latest/$file_name" \
        -o "$tmp_bin" || {
            rm -f "$tmp_bin" "$tmp_menu"
            echo "主程序下载失败，服务未更新。"
            return 1
        }

    log_step "下载最新菜单脚本"
    curl --fail --show-error --location --retry 3 \
        -H "Cache-Control: no-cache" \
        -H "Pragma: no-cache" \
        "https://raw.githubusercontent.com/$REPO/main/menu.sh" \
        -o "$tmp_menu" || {
            rm -f "$tmp_bin" "$tmp_menu"
            echo "菜单脚本下载失败，服务未更新。"
            return 1
        }

    log_step "替换文件并重启服务"
    chmod 755 "$tmp_bin" "$tmp_menu"
    if is_running; then
        systemctl stop sublink
    fi
    install -m 755 "$tmp_bin" "$BIN_PATH"
    install -m 755 "$tmp_menu" "$MENU_PATH"
    systemctl daemon-reload
    systemctl start sublink

    rm -f "$tmp_bin" "$tmp_menu"
    trap - RETURN
    echo "更新完成，当前版本: $("$BIN_PATH" --version)"
}

function start_sublink {
    if ! is_installed; then
        echo "当前未安装 SublinkX，请先安装。"
        return 1
    fi
    systemctl start sublink
    systemctl daemon-reload
    echo "服务已启动"
}

function stop_sublink {
    if ! is_installed; then
        echo "当前未安装 SublinkX。"
        return 1
    fi
    systemctl stop sublink
    systemctl daemon-reload
    echo "服务已停止"
}

function service_status {
    if ! is_installed; then
        echo "当前未安装 SublinkX。"
        return 1
    fi
    systemctl status sublink
}

function show_workdir {
    echo "运行目录: $INSTALL_DIR"
    echo "需要备份的目录为 db 和 template，logs 可按需备份。"
    cd "$INSTALL_DIR" || return 1
}

function change_port {
    if ! is_installed; then
        echo "当前未安装 SublinkX，请先安装。"
        return 1
    fi

    read -p "请输入新的端口号: " port
    echo "新的端口号: $port"
    parameter="run --port $port"
    if [ ! -f "$SERVICE_FILE" ]; then
        echo "服务文件不存在: $SERVICE_FILE"
        return 1
    fi

    if grep -q "run --port" "$SERVICE_FILE"; then
        echo "参数已存在，正在替换..."
        sed -i "s/--port [0-9]\+/--port $port/" "$SERVICE_FILE"
    else
        sed -i "/^ExecStart=/ s|$| $parameter|" "$SERVICE_FILE"
        echo "参数已添加到 ExecStart 行: $parameter"
    fi

    systemctl daemon-reload
    systemctl restart sublink
    echo "服务已重启。"
}

function reset_account {
    if ! is_installed; then
        echo "当前未安装 SublinkX，请先安装。"
        return 1
    fi

    read -p "请输入新的账号: " user
    read -p "请输入新的密码: " password
    cd "$INSTALL_DIR" || return 1
    "$BIN_PATH" setting --username "$user" --password "$password" &
    pid=$!
    wait "$pid"
    systemctl restart sublink
    echo "账号密码已重置，服务已重启。"
}

function Select {
    status="未安装"
    if is_installed; then
        if is_running; then
            status="已运行"
        else
            status="未运行"
        fi
    fi

    latest="$(latest_release 2>/dev/null || true)"
    version="$(current_version)"

    clear
    echo "SublinkX 管理菜单"
    echo "----------------"
    echo "最新版本: ${latest:-获取失败}"
    echo "当前版本: $version"
    echo "当前状态: $status"
    echo

    if is_installed; then
        echo "1. 卸载"
    else
        echo "1. 安装"
    fi
    echo "2. 更新"
    if is_running; then
        echo "3. 停止服务"
    else
        echo "3. 启动服务"
    fi
    echo "4. 查看服务状态"
    echo "5. 查看运行目录"
    echo "6. 修改端口"
    echo "7. 重置账号密码"
    echo "0. 退出"
    echo -n "请选择一个选项: "
    read option

    case $option in
        1)
            if is_installed; then
                uninstall_sublink
            else
                install_sublink
            fi
            ;;
        2)
            update_sublink
            ;;
        3)
            if is_running; then
                stop_sublink
            else
                start_sublink
            fi
            ;;
        4)
            service_status
            ;;
        5)
            show_workdir
            ;;
        6)
            change_port
            ;;
        7)
            reset_account
            ;;
        0)
            exit 0
            ;;
        *)
            echo "无效的选项，请重新选择。"
            ;;
    esac
}

Select
