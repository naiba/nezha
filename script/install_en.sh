#!/bin/sh

#========================================================
#   System Required: CentOS 7+ / Debian 8+ / Ubuntu 16+ / Alpine 3+ /
#   Arch has only been tested once, if there is any problem, please report with screenshots Dysf888@pm.me
#   Description: Nezha Monitoring Install Script
#   Github: https://github.com/naiba/nezha
#========================================================

NZ_BASE_PATH="/opt/nezha"
NZ_DASHBOARD_PATH="${NZ_BASE_PATH}/dashboard"
NZ_AGENT_PATH="${NZ_BASE_PATH}/agent"
NZ_DASHBOARD_SERVICE="/etc/systemd/system/nezha-dashboard.service"
NZ_DASHBOARD_SERVICERC="/etc/init.d/nezha-dashboard"
NZ_VERSION="v0.20.3"

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'
export PATH="$PATH:/usr/local/bin"

os_arch=""
[ -e /etc/os-release ] && grep -i "PRETTY_NAME" /etc/os-release | grep -qi "alpine" && os_alpine='1'

sudo() {
    myEUID=$(id -ru)
    if [ "$myEUID" -ne 0 ]; then
        if command -v sudo > /dev/null 2>&1; then
            command sudo "$@"
        else
            err "ERROR: sudo is not installed on the system, the action cannot be proceeded."
            exit 1
        fi
    else
        "$@"
    fi
}

check_systemd() {
    if [ "$os_alpine" != 1 ] && ! command -v systemctl >/dev/null 2>&1; then
        echo "System not supported: systemctl not found"
        exit 1
    fi
}

err() {
    printf "${red}%s${plain}\n" "$*" >&2
}

success() {
    printf "${green}%s${plain}\n" "$*"
}

info() {
    printf "${yellow}%s${plain}\n" "$*"
}

geo_check() {
    api_list="https://blog.cloudflare.com/cdn-cgi/trace https://dash.cloudflare.com/cdn-cgi/trace https://developers.cloudflare.com/cdn-cgi/trace"
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

pre_check() {
    umask 077

    ## os_arch
    if uname -m | grep -q 'x86_64'; then
        os_arch="amd64"
    elif uname -m | grep -q 'i386\|i686'; then
        os_arch="386"
    elif uname -m | grep -q 'aarch64\|armv8b\|armv8l'; then
        os_arch="arm64"
    elif uname -m | grep -q 'arm'; then
        os_arch="arm"
    elif uname -m | grep -q 's390x'; then
        os_arch="s390x"
    elif uname -m | grep -q 'riscv64'; then
        os_arch="riscv64"
    fi

    ## China_IP
    if [ -z "$CN" ]; then
        geo_check
        if [ -n "$isCN" ]; then
            echo "According to the information provided by various geoip api, the current IP may be in China"
            printf "Will the installation be done with a Chinese Mirror? [Y/n] (Custom Mirror Input 3): "
            read -r input
            case $input in
            [yY][eE][sS] | [yY])
                echo "Use Chinese Mirror"
                CN=true
                ;;

            [nN][oO] | [nN])
                echo "Do Not Use Chinese Mirror"
                ;;

            [3])
                echo "Use Custom Mirror"
                printf "Please enter a custom image (e.g. :dn-dao-github-mirror.daocloud.io). If left blank, it won't be used: "
                read -r input
                case $input in
                *)
                    CUSTOM_MIRROR=$input
                    ;;
                esac

                ;;
            *)
                echo "Do Not Use Chinese Mirror"
                ;;
            esac
        fi
    fi

    if [ -n "$CUSTOM_MIRROR" ]; then
        GITHUB_RAW_URL="gitee.com/naibahq/nezha/raw/master"
        GITHUB_URL=$CUSTOM_MIRROR
        Get_Docker_URL="get.docker.com"
        Get_Docker_Argu=" -s docker --mirror Aliyun"
        Docker_IMG="registry.cn-shanghai.aliyuncs.com\/naibahq\/nezha-dashboard"
    else
        if [ -z "$CN" ]; then
            GITHUB_RAW_URL="raw.githubusercontent.com/naiba/nezha/master"
            GITHUB_URL="github.com"
            Get_Docker_URL="get.docker.com"
            Get_Docker_Argu=" "
            Docker_IMG="ghcr.io\/naiba\/nezha-dashboard"
        else
            GITHUB_RAW_URL="gitee.com/naibahq/nezha/raw/master"
            GITHUB_URL="gitee.com"
            Get_Docker_URL="get.docker.com"
            Get_Docker_Argu=" -s docker --mirror Aliyun"
            Docker_IMG="registry.cn-shanghai.aliyuncs.com\/naibahq\/nezha-dashboard"
        fi
    fi
}

installation_check() {
    if docker compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE_COMMAND="docker compose"
        if sudo $DOCKER_COMPOSE_COMMAND ls | grep -qw "$NZ_DASHBOARD_PATH/docker-compose.yaml" >/dev/null 2>&1; then
            NEZHA_IMAGES=$(sudo docker images --format "{{.Repository}}:{{.Tag}}" | grep -w "nezha-dashboard")
            if [ -n "$NEZHA_IMAGES" ]; then
                echo "Docker image with nezha-dashboard repository exists:"
                echo "$NEZHA_IMAGES"
                IS_DOCKER_NEZHA=1
                FRESH_INSTALL=0
                return
            else
                echo "No Docker images with the nezha-dashboard repository were found."
            fi
        fi
    elif command -v docker-compose >/dev/null 2>&1; then
        DOCKER_COMPOSE_COMMAND="docker-compose"
        if sudo $DOCKER_COMPOSE_COMMAND -f "$NZ_DASHBOARD_PATH/docker-compose.yaml" config >/dev/null 2>&1; then
            NEZHA_IMAGES=$(sudo docker images --format "{{.Repository}}:{{.Tag}}" | grep -w "nezha-dashboard")
            if [ -n "$NEZHA_IMAGES" ]; then
                echo "Docker image with nezha-dashboard repository exists:"
                echo "$NEZHA_IMAGES"
                IS_DOCKER_NEZHA=1
                FRESH_INSTALL=0
                return
            else
                echo "No Docker images with the nezha-dashboard repository were found."
            fi
        fi
    fi

    if [ -f "$NZ_DASHBOARD_PATH/app" ]; then
        IS_DOCKER_NEZHA=0
        FRESH_INSTALL=0
    fi
}

select_version() {
    if [ -z "$IS_DOCKER_NEZHA" ]; then
        info "Select your installation method(Input anything is ok if you are installing agent):"
        info "1. Docker"
        info "2. Standalone"
        while true; do
            printf "Please enter [1-2]: "
            read -r option
            case "${option}" in
                1)
                    IS_DOCKER_NEZHA=1
                    break
                    ;;
                2)
                    IS_DOCKER_NEZHA=0
                    break
                    ;;
                *)
                    err "Please enter the correct number [1-2]"
                    ;;
            esac
        done
    fi
}

update_script() {
    echo "> Update Script"

    #curl -sL https://${GITHUB_RAW_URL}/script/install_en.sh -o /tmp/nezha.sh
    #new_version=$(grep "NZ_VERSION" /tmp/nezha.sh | head -n 1 | awk -F "=" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')
    #if [ -z "$new_version" ]; then
    #    echo "Script failed to get, please check if the network can link https://${GITHUB_RAW_URL}/script/install.sh"
    #    return 1
    #fi
    #echo "The current latest version is: ${new_version}"
    if [ -z "$CN" ]; then
        curl -sL https://raw.githubusercontent.com/nezhahq/scripts/main/install_en.sh -o /tmp/nezha.sh
    else
        curl -sL https://gitee.com/naibahq/scripts/raw/main/install_en.sh -o /tmp/nezha.sh
    fi
    mv -f /tmp/nezha.sh ./nezha.sh && chmod a+x ./nezha.sh

    echo "Execute new script after 3s"
    sleep 3s
    clear
    exec ./nezha.sh
    exit 0
}

before_show_menu() {
    echo && info "* Press Enter to return to the main menu *" && read temp
    show_menu
}

install_base() {
    (command -v curl >/dev/null 2>&1 && command -v wget >/dev/null 2>&1 && command -v unzip >/dev/null 2>&1 && command -v getenforce >/dev/null 2>&1) ||
        (install_soft curl wget unzip)
}

install_arch() {
    info "Archlinux needs to add nezha-agent user to install libselinux. It will be deleted automatically after installation. It is recommended to check manually"
    read -r -p "Do you need to install libselinux? [Y/n] " input
    case $input in
    [yY][eE][sS] | [yY])
        useradd -m nezha-agent
        sed -i "$ a\nezha-agent ALL=(ALL ) NOPASSWD:ALL" /etc/sudoers
        sudo -iu nezha-agent bash -c 'gpg --keyserver keys.gnupg.net --recv-keys 4695881C254508D1;
                                        cd /tmp; git clone https://aur.archlinux.org/libsepol.git; cd libsepol; makepkg -si --noconfirm --asdeps; cd ..;
                                        git clone https://aur.archlinux.org/libselinux.git; cd libselinux; makepkg -si --noconfirm; cd ..;
                                        rm -rf libsepol libselinux'
        sed -i '/nezha-agent/d' /etc/sudoers && sleep 30s && killall -u nezha-agent && userdel -r nezha-agent
        echo -e "${red}Info: ${plain}user nezha-agent has been deleted, Be sure to check it manually!\n"
        ;;
    [nN][oO] | [nN])
        echo "Libselinux will not be installed"
        ;;
    *)
        echo "Libselinux will not be installed"
        exit 0
        ;;
    esac
}

install_soft() {
    (command -v yum >/dev/null 2>&1 && sudo yum makecache && sudo yum install "$@" selinux-policy -y) ||
        (command -v apt >/dev/null 2>&1 && sudo apt update && sudo apt install "$@" selinux-utils -y) ||
        (command -v pacman >/dev/null 2>&1 && sudo pacman -Syu "$@" base-devel --noconfirm && install_arch) ||
        (command -v apt-get >/dev/null 2>&1 && sudo apt-get update && sudo apt-get install "$@" selinux-utils -y) ||
        (command -v apk >/dev/null 2>&1 && sudo apk update && sudo apk add "$@" -f)
}

install_dashboard() {
    check_systemd
    install_base

    echo "> Install Dashboard"

    # Nezha Monitoring Folder
    if [ ! "$FRESH_INSTALL" = 0 ]; then
        sudo mkdir -p $NZ_DASHBOARD_PATH
    else
        echo "You may have already installed the dashboard, repeated installation will overwrite the data, please pay attention to backup."
        printf "Exit the installation? [Y/n] "
        read -r input
        case $input in
        [yY][eE][sS] | [yY])
            echo "Exit the installation."
            exit 0
            ;;
        [nN][oO] | [nN])
            echo "Continue."
            ;;
        *)
            echo "Exit the installation."
            exit 0
            ;;
        esac
    fi

    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        install_dashboard_docker
    elif [ "$IS_DOCKER_NEZHA" = 0 ]; then
        install_dashboard_standalone
    fi

    modify_dashboard_config 0

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

install_dashboard_docker() {
    if [ ! "$FRESH_INSTALL" = 0 ]; then
        if ! command -v docker >/dev/null 2>&1; then
            echo "Installing Docker"
            if [ "$os_alpine" != 1 ]; then
                if ! curl -sL https://${Get_Docker_URL} | sudo bash -s "${Get_Docker_Argu}"; then
                    err "Script failed to get, please check if the network can link ${Get_Docker_URL}"
                    return 0
                fi
                sudo systemctl enable docker.service
                sudo systemctl start docker.service
            else
                sudo apk add docker docker-compose
                sudo rc-update add docker
                sudo rc-service docker start
            fi
            success "Docker installed successfully"
            installation_check
        fi
    fi
}

install_dashboard_standalone() {
    if [ ! -d "${NZ_DASHBOARD_PATH}/resource/template/theme-custom" ] || [ ! -d "${NZ_DASHBOARD_PATH}/resource/static/custom" ]; then
        sudo mkdir -p "${NZ_DASHBOARD_PATH}/resource/template/theme-custom" "${NZ_DASHBOARD_PATH}/resource/static/custom" >/dev/null 2>&1
    fi
}

selinux() {
    #Check SELinux
    if command -v getenforce >/dev/null 2>&1; then
        if getenforce | grep '[Ee]nfor'; then
            echo "SELinux running, closing now!"
            sudo setenforce 0 >/dev/null 2>&1
            find_key="SELINUX="
            sudo sed -ri "/^$find_key/c${find_key}disabled" /etc/selinux/config
        fi
    fi
}

install_agent() {
    install_base
    selinux

    echo "> Install Agent"

    echo "Obtaining Agent version number"


    _version=$(curl -m 10 -sL "https://api.github.com/repos/nezhahq/agent/releases/latest" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')
    if [ -z "$_version" ]; then
        _version=$(curl -m 10 -sL "https://gitee.com/api/v5/repos/naibahq/agent/releases/latest" | awk -F '"' '{for(i=1;i<=NF;i++){if($i=="tag_name"){print $(i+2)}}}')
    fi
    if [ -z "$_version" ]; then
        _version=$(curl -m 10 -sL "https://fastly.jsdelivr.net/gh/nezhahq/agent/" | grep "option\.value" | awk -F "'" '{print $2}' | sed 's/nezhahq\/agent@/v/g')
    fi
    if [ -z "$_version" ]; then
        _version=$(curl -m 10 -sL "https://gcore.jsdelivr.net/gh/nezhahq/agent/" | grep "option\.value" | awk -F "'" '{print $2}' | sed 's/nezhahq\/agent@/v/g')
    fi

    if [ -z "$_version" ]; then
        err "Fail to obtain agent version, please check if the network can link https://api.github.com/repos/nezhahq/agent/releases/latest"
        return 1
    else
        echo "The current latest version is: ${_version}"
    fi

    # Nezha Monitoring Folder
    sudo mkdir -p $NZ_AGENT_PATH

    echo "Downloading Agent"
    if [ -z "$CN" ]; then
        NZ_AGENT_URL="https://${GITHUB_URL}/nezhahq/agent/releases/download/${_version}/nezha-agent_linux_${os_arch}.zip"
    else
        NZ_AGENT_URL="https://${GITHUB_URL}/naibahq/agent/releases/download/${_version}/nezha-agent_linux_${os_arch}.zip"
    fi

    _cmd="wget -t 2 -T 60 -O nezha-agent_linux_${os_arch}.zip $NZ_AGENT_URL >/dev/null 2>&1"
    if ! eval "$_cmd"; then
        err "Fail to download agent, please check if the network can link ${GITHUB_URL}"
        return 1
    fi

    sudo unzip -qo nezha-agent_linux_${os_arch}.zip &&
        sudo mv nezha-agent $NZ_AGENT_PATH &&
        sudo rm -rf nezha-agent_linux_${os_arch}.zip README.md

    if [ $# -ge 3 ]; then
        modify_agent_config "$@"
    else
        modify_agent_config 0
    fi

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

modify_agent_config() {
    echo "> Modify Agent Configuration"

    if [ $# -lt 3 ]; then
        echo "Please add Agent in the Dashboard first, record the secret"
            printf "Please enter a domain that resolves to the IP where Dashboard is located (no CDN): "
            read -r nz_grpc_host
            printf "Please enter Dashboard RPC port (default 5555): "
            read -r nz_grpc_port
            printf "Please enter the Agent secret: "
            read -r nz_client_secret
            printf "Do you want to enable SSL/TLS encryption for the gRPC port (--tls)? Press [y] if yes, the default is not required, and users can press Enter to skip if you don't understand: "
            read -r nz_grpc_proxy
        echo "${nz_grpc_proxy}" | grep -qiw 'Y' && args='--tls'
        if [ -z "$nz_grpc_host" ] || [ -z "$nz_client_secret" ]; then
            err "All options cannot be empty"
            before_show_menu
            return 1
        fi
        if [ -z "$nz_grpc_port" ]; then
            nz_grpc_port=5555
        fi
    else
        nz_grpc_host=$1
        nz_grpc_port=$2
        nz_client_secret=$3
        shift 3
        if [ $# -gt 0 ]; then
            args="$*"
        fi
    fi

    _cmd="sudo ${NZ_AGENT_PATH}/nezha-agent service install -s $nz_grpc_host:$nz_grpc_port -p $nz_client_secret $args >/dev/null 2>&1"

    if ! eval "$_cmd"; then
        sudo "${NZ_AGENT_PATH}"/nezha-agent service uninstall >/dev/null 2>&1
        sudo "${NZ_AGENT_PATH}"/nezha-agent service install -s "$nz_grpc_host:$nz_grpc_port" -p "$nz_client_secret" "$args" >/dev/null 2>&1
    fi
    
    success "Agent configuration modified successfully, please wait for agent self-restart to take effect"

    #if [[ $# == 0 ]]; then
    #    before_show_menu
    #fi
}

modify_dashboard_config() {
    echo "> Modify Dashboard Configuration"

    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        if [ -n "$DOCKER_COMPOSE_COMMAND" ]; then
            echo "Download Docker Script"
            _cmd="wget -t 2 -T 60 -O /tmp/nezha-docker-compose.yaml https://${GITHUB_RAW_URL}/script/docker-compose.yaml >/dev/null 2>&1"
            if ! eval "$_cmd"; then
                err "Script failed to get, please check if the network can link ${GITHUB_RAW_URL}"
                return 0
            fi
        else
            err "Please install docker-compose manually. https://docs.docker.com/compose/install/linux/"
            before_show_menu
        fi
    fi

    _cmd="wget -t 2 -T 60 -O /tmp/nezha-config.yaml https://${GITHUB_RAW_URL}/script/config.yaml >/dev/null 2>&1"
    if ! eval "$_cmd"; then
        err "Script failed to get, please check if the network can link ${GITHUB_RAW_URL}"
        return 0
    fi

    echo "About the GitHub Oauth2 application: create it at https://github.com/settings/developers, no review required, and fill in the http(s)://domain_or_IP/oauth2/callback"
        echo "(Not recommended) About the Gitee Oauth2 application: create it at https://gitee.com/oauth/applications, no auditing required, and fill in the http(s)://domain_or_IP/oauth2/callback"
        printf "Please enter the OAuth2 provider (github/gitlab/jihulab/gitee, default github): "
        read -r nz_oauth2_type
        printf "Please enter the Client ID of the Oauth2 application: "
        read -r nz_github_oauth_client_id
        printf "Please enter the Client Secret of the Oauth2 application: "
        read -r nz_github_oauth_client_secret
        printf "Please enter your GitHub/Gitee login name as the administrator, separated by commas: "
        read -r nz_admin_logins
        printf "Please enter the site title: "
        read -r nz_site_title
        printf "Please enter the site access port: (default 8008)"
        read -r nz_site_port
        printf "Please enter the RPC port to be used for Agent access: (default 5555)"
        read -r nz_grpc_port

    if [ -z "$nz_admin_logins" ] || [ -z "$nz_github_oauth_client_id" ] || [ -z "$nz_github_oauth_client_secret" ] || [ -z "$nz_site_title" ]; then
        err "All options cannot be empty"
        before_show_menu
        return 1
    fi

    if [ -z "$nz_site_port" ]; then
        nz_site_port=8008
    fi
    if [ -z "$nz_grpc_port" ]; then
        nz_grpc_port=5555
    fi
    if [ -z "$nz_oauth2_type" ]; then
        nz_oauth2_type=github
    fi

    sed -i "s/nz_oauth2_type/${nz_oauth2_type}/" /tmp/nezha-config.yaml
    sed -i "s/nz_admin_logins/${nz_admin_logins}/" /tmp/nezha-config.yaml
    sed -i "s/nz_grpc_port/${nz_grpc_port}/" /tmp/nezha-config.yaml
    sed -i "s/nz_github_oauth_client_id/${nz_github_oauth_client_id}/" /tmp/nezha-config.yaml
    sed -i "s/nz_github_oauth_client_secret/${nz_github_oauth_client_secret}/" /tmp/nezha-config.yaml
    sed -i "s/nz_language/zh-CN/" /tmp/nezha-config.yaml
    sed -i "s/nz_site_title/${nz_site_title}/" /tmp/nezha-config.yaml
    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        sed -i "s/nz_site_port/${nz_site_port}/" /tmp/nezha-docker-compose.yaml
        sed -i "s/nz_grpc_port/${nz_grpc_port}/g" /tmp/nezha-docker-compose.yaml
        sed -i "s/nz_image_url/${Docker_IMG}/" /tmp/nezha-docker-compose.yaml
    elif [ "$IS_DOCKER_NEZHA" = 0 ]; then
        sed -i "s/80/${nz_site_port}/" /tmp/nezha-config.yaml
    fi

    sudo mkdir -p $NZ_DASHBOARD_PATH/data
    sudo mv -f /tmp/nezha-config.yaml ${NZ_DASHBOARD_PATH}/data/config.yaml
    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        sudo mv -f /tmp/nezha-docker-compose.yaml ${NZ_DASHBOARD_PATH}/docker-compose.yaml
    fi

    if [ "$IS_DOCKER_NEZHA" = 0 ]; then
        echo "Downloading service file"
        if [ "$os_alpine" != 1 ]; then
            _download="sudo wget -t 2 -T 60 -O $NZ_DASHBOARD_SERVICE https://${GITHUB_RAW_URL}/script/nezha-dashboard.service >/dev/null 2>&1"
            if ! eval "$_download"; then
                err "File failed to get, please check if the network can link ${GITHUB_RAW_URL}"
                return 0
            fi
        else
            _download="sudo wget -t 2 -T 60 -O $NZ_DASHBOARD_SERVICERC https://${GITHUB_RAW_URL}/script/nezha-dashboard >/dev/null 2>&1"
            if ! eval "$_download"; then
                err "File failed to get, please check if the network can link ${GITHUB_RAW_URL}"
                return 0
            fi
            sudo chmod +x $NZ_DASHBOARD_SERVICERC
        fi
    fi

    success "Dashboard configuration modified successfully, please wait for Dashboard self-restart to take effect"

    restart_and_update

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

restart_and_update() {
    echo "> Restart and Update Dashboard"

    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        _cmd="restart_and_update_docker"
    elif [ "$IS_DOCKER_NEZHA" = 0 ]; then
        _cmd="restart_and_update_standalone"
    fi

    if eval "$_cmd"; then
        success "Nezha Monitoring Restart Successful"
        info "Default Dashboard address: domain:site_access_port"
    else
        err "The restart failed, probably because the boot time exceeded two seconds, please check the log information later"
    fi

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

restart_and_update_docker() {
    sudo $DOCKER_COMPOSE_COMMAND -f ${NZ_DASHBOARD_PATH}/docker-compose.yaml pull
    sudo $DOCKER_COMPOSE_COMMAND -f ${NZ_DASHBOARD_PATH}/docker-compose.yaml down
    sudo $DOCKER_COMPOSE_COMMAND -f ${NZ_DASHBOARD_PATH}/docker-compose.yaml up -d
}

restart_and_update_standalone() {
    _version=$(curl -m 10 -sL "https://api.github.com/repos/naiba/nezha/releases/latest" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')
    if [ -z "$_version" ]; then
        _version=$(curl -m 10 -sL "https://gitee.com/api/v5/repos/naibahq/nezha/releases/latest" | awk -F '"' '{for(i=1;i<=NF;i++){if($i=="tag_name"){print $(i+2)}}}')
    fi
    if [ -z "$_version" ]; then
        _version=$(curl -m 10 -sL "https://fastly.jsdelivr.net/gh/naiba/nezha/" | grep "option\.value" | awk -F "'" '{print $2}' | sed 's/naiba\/nezha@/v/g')
    fi
    if [ -z "$_version" ]; then
        _version=$(curl -m 10 -sL "https://gcore.jsdelivr.net/gh/naiba/nezha/" | grep "option\.value" | awk -F "'" '{print $2}' | sed 's/naiba\/nezha@/v/g')
    fi

    if [ -z "$_version" ]; then
        err "Fail to obtain agent version, please check if the network can link https://api.github.com/repos/nezhahq/agent/releases/latest"
        return 1
    else
        echo "The current latest version is: ${_version}"
    fi

    if [ "$os_alpine" != 1 ]; then
        sudo systemctl daemon-reload
        sudo systemctl stop nezha-dashboard
    else
        sudo rc-service nezha-dashboard stop
    fi

    if [ -z "$CN" ]; then
        NZ_DASHBOARD_URL="https://${GITHUB_URL}/naiba/nezha/releases/download/${_version}/dashboard-linux-${os_arch}.zip"
    else
        NZ_DASHBOARD_URL="https://${GITHUB_URL}/naibahq/nezha/releases/download/${_version}/dashboard-linux-${os_arch}.zip"
    fi

    sudo wget -qO $NZ_DASHBOARD_PATH/app.zip "$NZ_DASHBOARD_URL" >/dev/null 2>&1 && sudo unzip -qq -o $NZ_DASHBOARD_PATH/app.zip -d $NZ_DASHBOARD_PATH && sudo mv $NZ_DASHBOARD_PATH/dashboard-linux-$os_arch $NZ_DASHBOARD_PATH/app && sudo rm $NZ_DASHBOARD_PATH/app.zip
    sudo chmod +x $NZ_DASHBOARD_PATH/app

    if [ "$os_alpine" != 1 ]; then
        sudo systemctl enable nezha-dashboard
        sudo systemctl restart nezha-dashboard
    else
        sudo rc-update add nezha-dashboard
        sudo rc-service nezha-dashboard restart
    fi
}

start_dashboard() {
    echo "> Start Dashboard"

    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        _cmd="start_dashboard_docker"
    elif [ "$IS_DOCKER_NEZHA" = 0 ]; then
        _cmd="start_dashboard_standalone"
    fi

    if eval "$_cmd"; then
        success "Nezha Monitoring Start Successful"
    else
        err "Failed to start, please check the log message later"
    fi

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

start_dashboard_docker() {
    sudo $DOCKER_COMPOSE_COMMAND -f ${NZ_DASHBOARD_PATH}/docker-compose.yaml up -d
}

start_dashboard_standalone() {
    if [ "$os_alpine" != 1 ]; then
        sudo systemctl start nezha-dashboard
    else
        sudo rc-service nezha-dashboard start
    fi
}

stop_dashboard() {
    echo "> Stop Dashboard"

    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        _cmd="stop_dashboard_docker"
    elif [ "$IS_DOCKER_NEZHA" = 0 ]; then
        _cmd="stop_dashboard_standalone"
    fi

    if eval "$_cmd"; then
        success "Nezha Monitoring Stop Successful"
    else
        err "Failed to stop, please check the log message later"
    fi

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

stop_dashboard_docker() {
    sudo $DOCKER_COMPOSE_COMMAND -f ${NZ_DASHBOARD_PATH}/docker-compose.yaml down
}

stop_dashboard_standalone() {
    if [ "$os_alpine" != 1 ]; then
        sudo systemctl stop nezha-dashboard
    else
        sudo rc-service nezha-dashboard stop
    fi
}

show_dashboard_log() {
    echo "> View Dashboard Log"

    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        show_dashboard_log_docker
    elif [ "$IS_DOCKER_NEZHA" = 0 ]; then
        show_dashboard_log_standalone
    fi

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

show_dashboard_log_docker() {
    sudo $DOCKER_COMPOSE_COMMAND -f ${NZ_DASHBOARD_PATH}/docker-compose.yaml logs -f
}

show_dashboard_log_standalone() {
    if [ "$os_alpine" != 1 ]; then
        sudo journalctl -xf -u nezha-dashboard.service
    else
        sudo tail -n 10 /var/log/nezha-dashboard.err
    fi
}

uninstall_dashboard() {
    echo "> Uninstall Dashboard"

    if [ "$IS_DOCKER_NEZHA" = 1 ]; then
        uninstall_dashboard_docker
    elif [ "$IS_DOCKER_NEZHA" = 0 ]; then
        uninstall_dashboard_standalone
    fi

    clean_all

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

uninstall_dashboard_docker() {
    sudo $DOCKER_COMPOSE_COMMAND -f ${NZ_DASHBOARD_PATH}/docker-compose.yaml down
    sudo rm -rf $NZ_DASHBOARD_PATH
    sudo docker rmi -f ghcr.io/naiba/nezha-dashboard >/dev/null 2>&1
    sudo docker rmi -f registry.cn-shanghai.aliyuncs.com/naibahq/nezha-dashboard >/dev/null 2>&1
}

uninstall_dashboard_standalone() {
    sudo rm -rf $NZ_DASHBOARD_PATH

    if [ "$os_alpine" != 1 ]; then
        sudo systemctl disable nezha-dashboard
        sudo systemctl stop nezha-dashboard
    else
        sudo rc-update del nezha-dashboard
        sudo rc-service nezha-dashboard stop
    fi

    if [ "$os_alpine" != 1 ]; then
        sudo rm $NZ_DASHBOARD_SERVICE
    else
        sudo rm $NZ_DASHBOARD_SERVICERC
    fi
}

show_agent_log() {
    echo "> View Agent Log"

    if [ "$os_alpine" != 1 ]; then
        sudo journalctl -xf -u nezha-agent.service
    else
        sudo tail -n 10 /var/log/nezha-agent.err
    fi

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

uninstall_agent() {
    echo "> Uninstall Agent"

    sudo ${NZ_AGENT_PATH}/nezha-agent service uninstall

    sudo rm -rf $NZ_AGENT_PATH
    clean_all

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

restart_agent() {
    echo "> Restart Agent"

    sudo ${NZ_AGENT_PATH}/nezha-agent service restart

    if [ $# = 0 ]; then
        before_show_menu
    fi
}

clean_all() {
    if [ -z "$(ls -A ${NZ_BASE_PATH})" ]; then
        sudo rm -rf ${NZ_BASE_PATH}
    fi
}

show_usage() {
    echo "Nezha Monitor Management Script Usage: "
    echo "--------------------------------------------------------"
    echo "./nezha.sh                            - Show Menu"
    echo "./nezha.sh install_dashboard          - Install Dashboard"
    echo "./nezha.sh modify_dashboard_config    - Modify Dashboard Configuration"
    echo "./nezha.sh start_dashboard            - Start Dashboard"
    echo "./nezha.sh stop_dashboard             - Stop Dashboard"
    echo "./nezha.sh restart_and_update         - Restart and Update the Dashboard"
    echo "./nezha.sh show_dashboard_log         - View Dashboard Log"
    echo "./nezha.sh uninstall_dashboard        - Uninstall Dashboard"
    echo "--------------------------------------------------------"
    echo "./nezha.sh install_agent              - Install Agent"
    echo "./nezha.sh modify_agent_config        - Modify Agent Configuration"
    echo "./nezha.sh show_agent_log             - View Agent Log"
    echo "./nezha.sh uninstall_agent            - Uninstall Agent"
    echo "./nezha.sh restart_agent              - Restart Agent"
    echo "./nezha.sh update_script              - Update Script"
    echo "--------------------------------------------------------"
}

show_menu() {
    printf "
    ${green}Nezha Monitor Management Script${plain} ${red}${NZ_VERSION}${plain}
    --- https://github.com/naiba/nezha ---
    ${green}1.${plain}  Install Dashboard
    ${green}2.${plain}  Modify Dashbaord Configuration
    ${green}3.${plain}  Start Dashboard
    ${green}4.${plain}  Stop Dashboard
    ${green}5.${plain}  Restart and Update Dashboard
    ${green}6.${plain}  View Dashboard Log
    ${green}7.${plain}  Uninstall Dashboard
    ————————————————-
    ${green}8.${plain}  Install Agent
    ${green}9.${plain}  Modify Agent Configuration
    ${green}10.${plain} View Agent Log
    ${green}11.${plain} Uninstall Agent
    ${green}12.${plain} Restart Agent
    ————————————————-
    ${green}13.${plain} Update Script
    ————————————————-
    ${green}0.${plain}  Exit Script
    "
    echo && printf "Please enter [0-13]: " && read -r num
    case "${num}" in
        0)
            exit 0
            ;;
        1)
            install_dashboard
            ;;
        2)
            modify_dashboard_config
            ;;
        3)
            start_dashboard
            ;;
        4)
            stop_dashboard
            ;;
        5)
            restart_and_update
            ;;
        6)
            show_dashboard_log
            ;;
        7)
            uninstall_dashboard
            ;;
        8)
            install_agent
            ;;
        9)
            modify_agent_config
            ;;
        10)
            show_agent_log
            ;;
        11)
            uninstall_agent
            ;;
        12)
            restart_agent
            ;;
        13)
            update_script
            ;;
        *)
            err "Please enter the correct number [0-13]"
            ;;
    esac
}

pre_check
installation_check

if [ $# -gt 0 ]; then
    case $1 in
        "install_dashboard")
            install_dashboard 0
            ;;
        "modify_dashboard_config")
            modify_dashboard_config 0
            ;;
        "start_dashboard")
            start_dashboard 0
            ;;
        "stop_dashboard")
            stop_dashboard 0
            ;;
        "restart_and_update")
            restart_and_update 0
            ;;
        "show_dashboard_log")
            show_dashboard_log 0
            ;;
        "uninstall_dashboard")
            uninstall_dashboard 0
            ;;
        "install_agent")
            shift
            if [ $# -ge 3 ]; then
                install_agent "$@"
            else
                install_agent 0
            fi
            ;;
        "modify_agent_config")
            modify_agent_config 0
            ;;
        "show_agent_log")
            show_agent_log 0
            ;;
        "uninstall_agent")
            uninstall_agent 0
            ;;
        "restart_agent")
            restart_agent 0
            ;;
        "update_script")
            update_script 0
            ;;
        *) show_usage ;;
    esac
else
    select_version
    show_menu
fi
