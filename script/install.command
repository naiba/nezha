#!/bin/bash

#========================================================
#   System Required: macOS 10.13+
#   Description: Nezha Agent Install Script (macOS)
#   Github: https://github.com/naiba/nezha
#========================================================

NZ_BASE_PATH="/opt/nezha"
NZ_AGENT_PATH="${NZ_BASE_PATH}/agent"

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'
export PATH=$PATH:/usr/local/bin

pre_check() {
    # check root
    [[ $EUID -ne 0 ]] && echo -e "${red}ERROR: ${plain} This script must be run with the root user!\n" && exit 1

    ## os_arch
    if [[ $(uname -m | grep 'x86_64') != "" ]]; then
        os_arch="amd64"
    elif [[ $(uname -m | grep 'arm64\|arm64e') != "" ]]; then
        os_arch="arm64"
    fi

    ## China_IP
    if [[ -z "${CN}" ]]; then
        if [[ $(curl -m 10 -s https://ipapi.co/json | grep 'China') != "" ]]; then
            echo "According to the information provided by ipapi.co, the current IP may be in China"
            read -e -r -p "Is the installation done with a Chinese Mirror? [Y/n] (Custom Mirror Input 3):" input
            case $input in
            [yY][eE][sS] | [yY])
                echo "Use Chinese Mirror"
                CN=true
                ;;

            [nN][oO] | [nN])
                echo "No Use Chinese Mirror"
                ;;

            [3])
                echo "Use Custom Mirror"
                read -e -r -p "Please enter a custom image (e.g. :dn-dao-github-mirror.daocloud.io), leave blank to nouse: " input
                case $input in
                *)
                    CUSTOM_MIRROR=$input
                    ;;
                esac

                ;;
            *)
                echo "No Use Chinese Mirror"
                ;;
            esac
        fi
    fi

    if [[ -n "${CUSTOM_MIRROR}" ]]; then
        GITHUB_RAW_URL="gitee.com/naibahq/nezha/raw/master"
        GITHUB_URL=$CUSTOM_MIRROR
    else
        if [[ -z "${CN}" ]]; then
            GITHUB_RAW_URL="raw.githubusercontent.com/naiba/nezha/master"
            GITHUB_URL="github.com"
            Get_Docker_URL="get.docker.com"
            Get_Docker_Argu=" "
            Docker_IMG="ghcr.io\/naiba\/nezha-dashboard"
        else
            GITHUB_RAW_URL="gitee.com/naibahq/nezha/raw/master"
            GITHUB_URL="github.com"
            Get_Docker_URL="get.docker.com"
            Get_Docker_Argu=" -s docker --mirror Aliyun"
            Docker_IMG="registry.cn-shanghai.aliyuncs.com\/naibahq\/nezha-dashboard"
        fi
    fi
}

before_show_menu() {
    echo && echo -n -e "${yellow}* Press Enter to return to the main menu *${plain}" && read temp
    show_menu
}

install_agent() {
    echo -e "> Install Nezha Agent"

    echo -e "Obtaining Agent version"

    local version=$(curl -m 10 -sL "https://api.github.com/repos/nezhahq/agent/releases/latest" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')
    if [ ! -n "$version" ]; then
        version=$(curl -m 10 -sL "https://fastly.jsdelivr.net/gh/nezhahq/agent/" | grep "option\.value" | awk -F "'" '{print $2}' | sed 's/nezhahq\/agent@/v/g')
    fi
    if [ ! -n "$version" ]; then
        version=$(curl -m 10 -sL "https://gcore.jsdelivr.net/gh/nezhahq/agent/" | grep "option\.value" | awk -F "'" '{print $2}' | sed 's/nezhahq\/agent@/v/g')
    fi

    if [ ! -n "$version" ]; then
        echo -e "Fail to obtaine agent version, please check if the network can link https://api.github.com/repos/nezhahq/agent/releases/latest"
        return 0
    else
        echo -e "The current latest version is: ${version}"
    fi

    # Nezha Agent Folder
    mkdir -p $NZ_AGENT_PATH
    chmod -R 777 $NZ_AGENT_PATH

    echo -e "Downloading Agent"
    curl -o nezha-agent_darwin_${os_arch}.zip -L -f --retry 2 --retry-max-time 60 https://${GITHUB_URL}/nezhahq/agent/releases/download/${version}/nezha-agent_darwin_${os_arch}.zip >/dev/null 2>&1
    if [[ $? != 0 ]]; then
        echo -e "${red}Fail to download agent, please check if the network can link ${GITHUB_URL}${plain}"
        return 0
    fi

    unzip -qo nezha-agent_darwin_${os_arch}.zip &&
        mv nezha-agent $NZ_AGENT_PATH &&
        rm -rf nezha-agent_darwin_${os_arch}.zip README.md

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

    if [ $# -lt 3 ]; then
        echo "Please add Agent in the admin panel first, record the secret" &&
            read -ep "Please enter a domain that resolves to the IP where the panel is located (no CDN sets): " nz_grpc_host &&
            read -ep "Please enter the panel RPC port (default 5555): " nz_grpc_port &&
            read -ep "Please enter the Agent secret: " nz_client_secret &&
            read -ep "Do you want to enable SSL/TLS encryption for the gRPC port (--tls)? Press [y] if yes, the default is not required, and users can press Enter to skip if you don't understand: " nz_grpc_proxy
        grep -qiw 'Y' <<<"${nz_grpc_proxy}" && args='--tls'
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
        shift 3
        if [ $# -gt 0 ]; then
            args=" $*"
        fi
    fi

    ${NZ_AGENT_PATH}/nezha-agent service install -s "$nz_grpc_host:$nz_grpc_port" -p $nz_client_secret $args >/dev/null 2>&1

    if [ $? -ne 0 ]; then
        ${NZ_AGENT_PATH}/nezha-agent service uninstall >/dev/null 2>&1
        ${NZ_AGENT_PATH}/nezha-agent service install -s "$nz_grpc_host:$nz_grpc_port" -p $nz_client_secret $args >/dev/null 2>&1
    fi

    echo -e "Agent configuration ${green} modified successfully, please wait for agent self-restart to take effect${plain}"

    #if [[ $# == 0 ]]; then
    #    before_show_menu
    #fi
}

show_agent_log() {
    echo -e "> > View Agent Log"

    tail -n 10 /var/log/nezha-agent.err.log

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

uninstall_agent() {
    echo -e "> Uninstall Agent"

    ${NZ_AGENT_PATH}/nezha-agent service uninstall

    rm -rf $NZ_AGENT_PATH
    clean_all

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

restart_agent() {
    echo -e "> Restart Agent"

    ${NZ_AGENT_PATH}/nezha-agent service restart

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
    echo "Nezha Agent Management Script Usage: "
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
    ${green}Nezha Agent Management Script${plain} ${red}macOS${plain}
    --- https://github.com/naiba/nezha ---
    ${green}1.${plain}  Install Agent
    ${green}2.${plain}  Modify Agent Configuration
    ${green}3.${plain}  View Agent Log
    ${green}4.${plain}  Uninstall Agent
    ${green}5.${plain}  Restart Agent
    ————————————————-
    ${green}0.${plain}  Exit Script
    "
    echo && read -ep "Please enter [0-5]: " num
    case "${num}" in
        0)
            exit 0
            ;;
        1)
            install_agent
            ;;
        2)
            modify_agent_config
            ;;
        3)
            show_agent_log
            ;;
        4)
            uninstall_agent
            ;;
        5)
            restart_agent
            ;;
        *)
            echo -e "${red}Please enter the correct number [0-5]${plain}"
            ;;
    esac
}

pre_check

if [[ $# > 0 ]]; then
    case $1 in
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
        *) show_usage ;;
    esac
else
    show_menu
fi