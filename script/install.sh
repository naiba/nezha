#!/bin/sh

#========================================================
# v0 分支脚本强制重定向至新仓库
#========================================================

# 判断是否应使用中国镜像

geo_check() {
    api_list="https://blog.cloudflare.com/cdn-cgi/trace https://developers.cloudflare.com/cdn-cgi/trace"
    ua="Mozilla/5.0 (X11; Linux x86_64; rv:60.0) Gecko/20100101 Firefox/81.0"
    set -- "$api_list"
    for url in $api_list; do
        text="$(curl -A "$ua" -m 10 -s "$url")"
        endpoint="$(echo "$text" | sed -n 's/.*h=\([^ ]*\).*/\1/p')"
        if echo "$text" | grep -qw 'CN'; then
            isCN=true
            break
        elif echo "$url" | grep -q "$endpoint"; then
            break
        fi
    done
}

# 向用户确认是否使用中国镜像
geo_check

if [ "$isCN" = true ]; then
    read -p "检测到您的IP可能来自中国大陆，是否使用中国镜像? [y/n] " choice
    case "$choice" in
        y|Y)
            echo "将使用中国镜像..."
            USE_CN_MIRROR=true
            ;;
        n|N)
            echo "将使用国际镜像..."
            USE_CN_MIRROR=false
            ;;
        *)
            echo "输入无效,将使用国际镜像..."
            USE_CN_MIRROR=false
            ;;
    esac
else
    USE_CN_MIRROR=false
fi

if [ "$USE_CN_MIRROR" = true ]; then
    shell_url="https://gitee.com/naibahq/scripts/raw/v0/install.sh"
else
    shell_url="https://raw.githubusercontent.com/nezhahq/scripts/refs/heads/v0/install.sh"
fi


# 新地址 https://raw.githubusercontent.com/nezhahq/scripts/refs/heads/v0/install.sh
if command -v wget >/dev/null 2>&1; then
    wget -O nezha_v0.sh "$shell_url"
elif command -v curl >/dev/null 2>&1; then
    curl -o nezha_v0.sh "$shell_url"
else
    echo "错误: 未找到 wget 或 curl，请安装其中任意一个后再试"
    exit 1
fi

chmod +x nezha_v0.sh

# 携带原参数运行新脚本
exec ./nezha_v0.sh "$@"
