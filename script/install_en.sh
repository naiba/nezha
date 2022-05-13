#!/bin/bash

#========================================================
#   System Required: CentOS 7+ / Debian 8+ / Ubuntu 16+ /
#     Arch not tested
#   Description: Nezha Monitoring Install Script
#   Github: https://github.com/naiba/nezha
#========================================================

NZ_BASE_PATH="/opt/nezha"
NZ_DASHBOARD_PATH="${NZ_BASE_PATH}/dashboard"
NZ_AGENT_PATH="${NZ_BASE_PATH}/agent"
NZ_AGENT_SERVICE="/etc/systemd/system/nezha-agent.service"
NZ_VERSION="v0.10.1"

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'
export PATH=$PATH:/usr/local/bin

os_arch=""

pre_check() {
    command -v systemctl >/dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo "This system is not supported: systemctl not found"
        exit 1
    fi

    # check root
    [[ $EUID -ne 0 ]] && echo -e "${red}ERROR: ${plain} This script must be run with the root user!\n" && exit 1

    ## os_arch
    if [[ $(uname -m | grep 'x86_64') != "" ]]; then
        os_arch="amd64"
    elif [[ $(uname -m | grep 'i386\|i686') != "" ]]; then
        os_arch="386"
    elif [[ $(uname -m | grep 'aarch64\|armv8b\|armv8l') != "" ]]; then
        os_arch="arm64"
    elif [[ $(uname -m | grep 'arm') != "" ]]; then
        os_arch="arm"
    elif [[ $(uname -m | grep 's390x') != "" ]]; then
        os_arch="s390x"
    elif [[ $(uname -m | grep 'riscv64') != "" ]]; then
        os_arch="riscv64"
    fi

    GITHUB_RAW_URL="raw.githubusercontent.com/naiba/nezha/master"
    GITHUB_URL="github.com"
    Get_Docker_URL="get.docker.com"
    Get_Docker_Argu=" "
    Docker_IMG="ghcr.io\/naiba\/nezha-dashboard"
}

confirm() {
    if [[ $# > 1 ]]; then
        echo && read -e -p "$1 [Default$2]: " temp
        if [[ x"${temp}" == x"" ]]; then
            temp=$2
        fi
    else
        read -e -p "$1 [y/n]: " temp
    fi
    if [[ x"${temp}" == x"y" || x"${temp}" == x"Y" ]]; then
        return 0
    else
        return 1
    fi
}

update_script() {
    echo -e "> Update Script"

    curl -sL https://${GITHUB_RAW_URL}/script/install.sh -o /tmp/nezha.sh
    new_version=$(cat /tmp/nezha.sh | grep "NZ_VERSION" | head -n 1 | awk -F "=" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')
    if [ ! -n "$new_version" ]; then
        echo -e "Script failed to get, please check if the network can link https://${GITHUB_RAW_URL}/script/install.sh"
        return 1
    fi
    echo -e "The current latest version is: ${new_version}"
    mv -f /tmp/nezha.sh ./nezha.sh && chmod a+x ./nezha.sh

    echo -e "Execute new script after 3s"
    sleep 3s
    clear
    exec ./nezha.sh
    exit 0
}

before_show_menu() {
    echo && echo -n -e "${yellow}* Press Enter to return to the main menu *${plain}" && read temp
    show_menu
}

install_base() {
    (command -v git >/dev/null 2>&1 && command -v curl >/dev/null 2>&1 && command -v wget >/dev/null 2>&1 && command -v unzip >/dev/null 2>&1 && command -v getenforce >/dev/null 2>&1) ||
        (install_soft curl wget git unzip)
}

install_soft() {
    # The official Arch library does not contain selinux components
    (command -v yum >/dev/null 2>&1 && yum install $* selinux-policy -y) ||
        (command -v apt >/dev/null 2>&1 && apt install $* selinux-utils -y) ||
        (command -v pacman >/dev/null 2>&1 && pacman -Syu $*) ||
        (command -v apt-get >/dev/null 2>&1 && apt-get install $* selinux-utils -y)
}

install_dashboard() {
    install_base

    echo -e "> Install Panel"

    # Nezha Monitoring Folder
    if [ ! -d "${NZ_DASHBOARD_PATH}" ]; then
        mkdir -p $NZ_DASHBOARD_PATH
    else
        echo "You may have already installed the dashboard, repeated installation will overwrite the data, please pay attention to backup."
        read -e -r -p "Exit the installation? [Y/n] " input
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
    chmod 777 -R $NZ_DASHBOARD_PATH

    command -v docker >/dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo -e "Installing Docker"
        bash <(curl -sL https://${Get_Docker_URL}) ${Get_Docker_Argu} >/dev/null 2>&1
        if [[ $? != 0 ]]; then
            echo -e "${red}Script failed to get, please check if the network can link ${Get_Docker_URL}${plain}"
            return 0
        fi
        systemctl enable docker.service
        systemctl start docker.service
        echo -e "${green}Docker${plain} installed successfully"
    fi

    command -v docker-compose >/dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo -e "Installing Docker Compose"
        wget -O /usr/local/bin/docker-compose "https://${GITHUB_URL}/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" >/dev/null 2>&1
        if [[ $? != 0 ]]; then
            echo -e "${red}Script failed to get, please check if the network can link ${GITHUB_URL}${plain}"
            return 0
        fi
        chmod +x /usr/local/bin/docker-compose
        echo -e "${green}Docker Compose${plain} installed successfully"
    fi

    modify_dashboard_config 0

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

selinux(){
    #Check SELinux
    getenforce | grep enfor
    if [ $? -eq 0 ];then
        echo -e "SELinux running，closing now！" 
        setenforce 0 &>/dev/null
        find_key="SELINUX="
        sed -ri "/^$find_key/c${find_key}disabled" /etc/selinux/config
    fi
}

install_agent() {
    install_base
    selinux

    echo -e "> Install Nezha Agent"

    echo -e "Obtaining Agent version"

    local version=$(curl -m 10 -sL "https://api.github.com/repos/naiba/nezha/releases/latest" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')
    if [ ! -n "$version" ]; then
        version=$(curl -m 10 -sL "https://fastly.jsdelivr.net/gh/naiba/nezha/" | grep "option\.value" | awk -F "'" '{print $2}' | sed 's/naiba\/nezha@/v/g')
    fi
    if [ ! -n "$version" ]; then
        version=$(curl -m 10 -sL "https://gcore.jsdelivr.net/gh/naiba/nezha/" | grep "option\.value" | awk -F "'" '{print $2}' | sed 's/naiba\/nezha@/v/g')
    fi

    if [ ! -n "$version" ]; then
        echo -e "Fail to obtaine agent version, please check if the network can link https://api.github.com/repos/naiba/nezha/releases/latest"
        return 0
    else
        echo -e "The current latest version is: ${version}"
    fi

    # Nezha Monitoring Folder
    mkdir -p $NZ_AGENT_PATH
    chmod 777 -R $NZ_AGENT_PATH

    echo -e "Downloading Agent"
    wget -O nezha-agent_linux_${os_arch}.zip https://${GITHUB_URL}/naiba/nezha/releases/download/${version}/nezha-agent_linux_${os_arch}.zip >/dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo -e "${red}Fail to download agent, please check if the network can link ${GITHUB_URL}${plain}"
        return 0
    fi

    unzip -qo nezha-agent_linux_${os_arch}.zip &&
        mv nezha-agent $NZ_AGENT_PATH &&
        rm -rf nezha-agent_linux_${os_arch}.zip README.md

    if [ $# -ge 3 ]; then
        modify_agent_config "$@"
    else
        modify_agent_config 0
    fi

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

modify_agent_config() {
    echo -e "> Modify Agent Configuration"

    wget -O $NZ_AGENT_SERVICE https://${GITHUB_RAW_URL}/script/nezha-agent.service >/dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo -e "${red}Fail to download service, please check if the network can link ${GITHUB_RAW_URL}${plain}"
        return 0
    fi

    if [ $# -lt 3 ]; then
        echo "Please add Agent in the admin panel first, record the secret" &&
            read -ep "Please enter a domain that resolves to the IP where the panel is located (no CDN sets): " nz_grpc_host &&
            read -ep "Please enter the panel RPC port: (5555)" nz_grpc_port &&
            read -ep "Please enter the Agent secret: " nz_client_secret
        if [[ -z "${nz_grpc_host}" || -z "${nz_client_secret}" ]]; then
            echo -e "${red}All options cannot be empty${plain}"
            before_show_menu
            return 1
        fi
        if [[ -z "${nz_grpc_port}" ]]; then
            nz_grpc_port=5555
        fi
    else
        nz_grpc_host=$1
        nz_grpc_port=$2
        nz_client_secret=$3
    fi

    sed -i "s/nz_grpc_host/${nz_grpc_host}/" ${NZ_AGENT_SERVICE}
    sed -i "s/nz_grpc_port/${nz_grpc_port}/" ${NZ_AGENT_SERVICE}
    sed -i "s/nz_client_secret/${nz_client_secret}/" ${NZ_AGENT_SERVICE}

    shift 3
    if [ $# -gt 0 ]; then
        args=" $*"
        sed -i "/ExecStart/ s/$/${args}/" ${NZ_AGENT_SERVICE}
    fi

    echo -e "Agent configuration ${green} modified successfully, please wait for agent self-restart to take effect${plain}"

    systemctl daemon-reload
    systemctl enable nezha-agent
    systemctl restart nezha-agent

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

modify_dashboard_config() {
    echo -e "> Modify Panel Configuration"

    echo -e "Download Docker Script"
    wget -O ${NZ_DASHBOARD_PATH}/docker-compose.yaml https://${GITHUB_RAW_URL}/script/docker-compose.yaml >/dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo -e "${red}Script failed to get, please check if the network can link ${GITHUB_RAW_URL}${plain}"
        return 0
    fi

    mkdir -p $NZ_DASHBOARD_PATH/data

    wget -O ${NZ_DASHBOARD_PATH}/data/config.yaml https://${GITHUB_RAW_URL}/script/config.yaml >/dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo -e "${red}Script failed to get, please check if the network can link ${GITHUB_RAW_URL}${plain}"
        return 0
    fi

    echo "About the GitHub Oauth2 application: create it at https://github.com/settings/developers, no review required, and fill in the http(s)://domain_or_IP/oauth2/callback" &&
        echo "(Not recommended) About the Gitee Oauth2 application: create it at https://gitee.com/oauth/applications, no auditing required, and fill in the http(s)://domain_or_IP/oauth2/callback" &&
        read -ep "Please enter the OAuth2 provider (gitee/github, default github): " nz_oauth2_type &&
        read -ep "Please enter the Client ID of the Oauth2 application: " nz_github_oauth_client_id &&
        read -ep "Please enter the Client Secret of the Oauth2 application: " nz_github_oauth_client_secret &&
        read -ep "Please enter your GitHub/Gitee login name as the administrator, separated by commas: " nz_admin_logins &&
        read -ep "Please enter the site title: " nz_site_title &&
        read -ep "Please enter the site access port: (default 8008)" nz_site_port &&
        read -ep "Please enter the RPC port to be used for Agent access: (default 5555)" nz_grpc_port

    if [[ -z "${nz_admin_logins}" || -z "${nz_github_oauth_client_id}" || -z "${nz_github_oauth_client_secret}" || -z "${nz_site_title}" ]]; then
        echo -e "${red}All options cannot be empty${plain}"
        before_show_menu
        return 1
    fi

    if [[ -z "${nz_site_port}" ]]; then
        nz_site_port=8008
    fi
    if [[ -z "${nz_grpc_port}" ]]; then
        nz_grpc_port=5555
    fi
    if [[ -z "${nz_oauth2_type}" ]]; then
        nz_oauth2_type=github
    fi

    sed -i "s/nz_oauth2_type/${nz_oauth2_type}/" ${NZ_DASHBOARD_PATH}/data/config.yaml
    sed -i "s/nz_admin_logins/${nz_admin_logins}/" ${NZ_DASHBOARD_PATH}/data/config.yaml
    sed -i "s/nz_grpc_port/${nz_grpc_port}/" ${NZ_DASHBOARD_PATH}/data/config.yaml
    sed -i "s/nz_github_oauth_client_id/${nz_github_oauth_client_id}/" ${NZ_DASHBOARD_PATH}/data/config.yaml
    sed -i "s/nz_github_oauth_client_secret/${nz_github_oauth_client_secret}/" ${NZ_DASHBOARD_PATH}/data/config.yaml
    sed -i "s/nz_site_title/${nz_site_title}/" ${NZ_DASHBOARD_PATH}/data/config.yaml
    sed -i "s/nz_language/en-US/" ${NZ_DASHBOARD_PATH}/data/config.yaml
    sed -i "s/nz_site_port/${nz_site_port}/" ${NZ_DASHBOARD_PATH}/docker-compose.yaml
    sed -i "s/nz_grpc_port/${nz_grpc_port}/g" ${NZ_DASHBOARD_PATH}/docker-compose.yaml
    sed -i "s/nz_image_url/${Docker_IMG}/" ${NZ_DASHBOARD_PATH}/docker-compose.yaml


    echo -e "Dashboard configuration ${green} modified successfully, please wait for Dashboard self-restart to take effect${plain}"

    restart_and_update

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

restart_and_update() {
    echo -e "> Restart and Update the Panel"

    cd $NZ_DASHBOARD_PATH
    docker-compose pull
    docker-compose down
    docker-compose up -d
    if [[ $? == 0 ]]; then
        echo -e "${green}Nezha Monitoring Restart Successful${plain}"
        echo -e "Default panel address: ${yellow}domain:Site_access_port${plain}"
    else
        echo -e "${red}The restart failed, probably because the boot time exceeded two seconds, please check the log information later${plain}"
    fi

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

start_dashboard() {
    echo -e "> Start Panel"

    cd $NZ_DASHBOARD_PATH && docker-compose up -d
    if [[ $? == 0 ]]; then
        echo -e "${green}Nezha Monitoring Start Successful${plain}"
    else
        echo -e "${red}Failed to start, please check the log message later${plain}"
    fi

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

stop_dashboard() {
    echo -e "> Stop Panel"

    cd $NZ_DASHBOARD_PATH && docker-compose down
    if [[ $? == 0 ]]; then
        echo -e "${green}Nezha Monitoring Stop Successful${plain}"
    else
        echo -e "${red}Failed to stop, please check the log message later${plain}"
    fi

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

show_dashboard_log() {
    echo -e "> View Panel Log"

    cd $NZ_DASHBOARD_PATH && docker-compose logs -f

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

uninstall_dashboard() {
    echo -e "> Uninstall Panel"

    cd $NZ_DASHBOARD_PATH &&
        docker-compose down
    rm -rf $NZ_DASHBOARD_PATH
    docker rmi -f ghcr.io/naiba/nezha-dashboard > /dev/null 2>&1
    clean_all

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

show_agent_log() {
    echo -e "> View Agent Log"

    journalctl -xf -u nezha-agent.service

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

uninstall_agent() {
    echo -e "> Uninstall Agent"

    systemctl disable nezha-agent.service
    systemctl stop nezha-agent.service
    rm -rf $NZ_AGENT_SERVICE
    systemctl daemon-reload

    rm -rf $NZ_AGENT_PATH
    clean_all

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

restart_agent() {
    echo -e "> Restart Agent"

    systemctl restart nezha-agent.service

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

clean_all() {
    if [ -z "$(ls -A ${NZ_BASE_PATH})" ]; then
        rm -rf ${NZ_BASE_PATH}
    fi
}

show_usage() {
    echo "Nezha Monitor Management Script Usage: "
    echo "--------------------------------------------------------"
    echo "./nezha.sh                            - Show Menu"
    echo "./nezha.sh install_dashboard          - Install Panel"
    echo "./nezha.sh modify_dashboard_config    - Modify Panel Configuration"
    echo "./nezha.sh start_dashboard            - Start Panel"
    echo "./nezha.sh stop_dashboard             - Stop Panel"
    echo "./nezha.sh restart_and_update         - Restart and Update the Panel"
    echo "./nezha.sh show_dashboard_log         - View Panel Log"
    echo "./nezha.sh uninstall_dashboard        - Uninstall Panel"
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
    echo -e "
    ${green}Nezha Monitor Management Script${plain} ${red}${NZ_VERSION}${plain}
    --- https://github.com/naiba/nezha ---
    ${green}1.${plain}  Install Panel
    ${green}2.${plain}  Modify Panel Configuration
    ${green}3.${plain}  Start Panel
    ${green}4.${plain}  Stop Panel
    ${green}5.${plain}  Restart and Update the Panel
    ${green}6.${plain}  View Panel Log
    ${green}7.${plain}  Uninstall Panel
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
    echo && read -ep "Please enter [0-13]: " num

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
        echo -e "${red}Please enter the correct number [0-13]${plain}"
        ;;
    esac
}

pre_check

if [[ $# > 0 ]]; then
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
    show_menu
fi
