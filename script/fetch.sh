if [[ -z "${CN}" ]]; then
    GITHUB_RAW_URL="raw.githubusercontent.com/naiba/nezha/master"
else
    GITHUB_RAW_URL="cdn.jsdelivr.net/gh/naiba/nezha@master"
fi
mkdir -p /opt/nezha
chmod 777 /opt/nezha
curl -sSL https://${GITHUB_RAW_URL}/script/install.sh -o /opt/nezha/nezha.sh
chmod +x /opt/nezha/nezha.sh