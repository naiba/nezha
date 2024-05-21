#Get server and key
param($server, $key, $tls)
# Download latest release from github
if($PSVersionTable.PSVersion.Major -lt 5){
    Write-Host "Require PS >= 5,your PSVersion:"$PSVersionTable.PSVersion.Major -BackgroundColor DarkGreen -ForegroundColor White
    Write-Host "Refer to the community article and install manually! https://nyko.me/2020/12/13/nezha-windows-client.html" -BackgroundColor DarkRed -ForegroundColor Green
    exit
}
$agentrepo = "nezhahq/agent"
#  x86 or x64 or arm64
if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
        $file = "nezha-agent_windows_arm64.zip"
    } else {
        $file = "nezha-agent_windows_amd64.zip"
    }
}
else {
    $file = "nezha-agent_windows_386.zip"
}
$agentreleases = "https://api.github.com/repos/$agentrepo/releases"
#重复运行自动更新
if (Test-Path "C:\nezha\nezha-agent.exe") {
    Write-Host "Nezha monitoring already exists, delete and reinstall" -BackgroundColor DarkGreen -ForegroundColor White
    C:\nezha\nezha-agent.exe service uninstall
    Remove-Item "C:\nezha" -Recurse
}
#TLS/SSL
Write-Host "Determining latest nezha release" -BackgroundColor DarkGreen -ForegroundColor White
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$agenttag = (Invoke-WebRequest -Uri $agentreleases -UseBasicParsing | ConvertFrom-Json)[0].tag_name
if ([string]::IsNullOrWhiteSpace($agenttag)) {
    $optionUrl = "https://fastly.jsdelivr.net/gh/nezhahq/agent/"
    Try {
        $response = Invoke-WebRequest -Uri $optionUrl -UseBasicParsing -TimeoutSec 10
        if ($response.StatusCode -eq 200) {
            $versiontext = $response.Content | findstr /c:"option.value"
            $version = [regex]::Match($versiontext, "@(\d+\.\d+\.\d+)").Groups[1].Value
            $agenttag = "v" + $version
        }
    } Catch {
        $optionUrl = "https://gcore.jsdelivr.net/gh/nezhahq/agent/"
        $response = Invoke-WebRequest -Uri $optionUrl -UseBasicParsing -TimeoutSec 10
        if ($response.StatusCode -eq 200) {
            $versiontext = $response.Content | findstr /c:"option.value"
            $version = [regex]::Match($versiontext, "@(\d+\.\d+\.\d+)").Groups[1].Value
            $agenttag = "v" + $version
        }
    }
}
#Region判断
$ipapi= Invoke-RestMethod  -Uri "https://api.myip.com/" -UserAgent "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.1 (KHTML, like Gecko) Chrome/14.0.835.163 Safari/535.1"
$region=$ipapi.cc
echo $ipapi
if($region -ne "CN"){
$download = "https://github.com/$agentrepo/releases/download/$agenttag/$file"
Write-Host "Location:$region,connect directly!" -BackgroundColor DarkRed -ForegroundColor Green
}else{
$download = "https://github.com/$agentrepo/releases/download/$agenttag/$file"
Write-Host "Location:CN,use mirror address" -BackgroundColor DarkRed -ForegroundColor Green
}
echo $download
Invoke-WebRequest $download -OutFile "C:\nezha.zip"
#解压
Expand-Archive "C:\nezha.zip" -DestinationPath "C:\temp" -Force
if (!(Test-Path "C:\nezha")) { New-Item -Path "C:\nezha" -type directory }
#整理文件
Move-Item -Path "C:\temp\nezha-agent.exe" -Destination "C:\nezha\nezha-agent.exe"
#清理垃圾
Remove-Item "C:\nezha.zip"
Remove-Item "C:\temp" -Recurse
#安装部分
C:\nezha\nezha-agent.exe service install -s $server -p $key $tls
#enjoy
Write-Host "Enjoy It!" -BackgroundColor DarkGreen -ForegroundColor Red