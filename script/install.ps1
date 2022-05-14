#Get server and key
param($server, $key, $tls)
# Download latest release from github
$repo = "naiba/nezha"
#  x86 or x64
if ([System.Environment]::Is64BitOperatingSystem) {
    $file = "nezha-agent_windows_amd64.zip"
}
else {
    $file = "nezha-agent_windows_386.zip"
}
$releases = "https://api.github.com/repos/$repo/releases"
#重复运行自动更新
if (Test-Path "C:\nezha") {
    Write-Host "Nezha monitoring already exists, delete and reinstall" -BackgroundColor DarkGreen -ForegroundColor White
    C:/nezha/nssm.exe stop nezha
    C:/nezha/nssm.exe remove nezha
    Remove-Item "C:\nezha" -Recurse
}
#TLS/SSL
Write-Host "Determining latest nezha release" -BackgroundColor DarkGreen -ForegroundColor White
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$tag = (Invoke-WebRequest -Uri $releases -UseBasicParsing | ConvertFrom-Json)[0].tag_name
#Region判断
$ipapi= Invoke-RestMethod  -Uri "https://api.myip.com/" -UserAgent "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/535.1 (KHTML, like Gecko) Chrome/14.0.835.163 Safari/535.1"
$region=$ipapi.cc
echo $ipapi
if($region -ne "CN"){
$download = "https://github.com/$repo/releases/download/$tag/$file"
Write-Host "Overseas machine("$region") direct connection!" -BackgroundColor DarkRed -ForegroundColor Green
echo $download
}elseif($region -eq $null){
cls
$download = "https://ghproxy.com/github.com/$repo/releases/download/$tag/$file"
Write-Host "Error,Most of the time, it is caused by the domestic network environment,use ghproxy.com" -BackgroundColor DarkRed -ForegroundColor Green
echo $download
}else{
$download = "https://ghproxy.com/github.com/$repo/releases/download/$tag/$file"
Write-Host "China's servers will be downloaded using the image address" -BackgroundColor DarkRed -ForegroundColor Green
echo $download
}
Invoke-WebRequest $download -OutFile "C:\nezha.zip"
#使用nssm安装服务
Invoke-WebRequest "http://nssm.cc/release/nssm-2.24.zip" -OutFile "C:\nssm.zip"
#解压
Expand-Archive "C:\nezha.zip" -DestinationPath "C:\temp" -Force
Expand-Archive "C:\nssm.zip" -DestinationPath "C:\temp" -Force
if (!(Test-Path "C:\nezha")) { New-Item -Path "C:\nezha" -type directory }
#整理文件
Move-Item -Path "C:\temp\nezha-agent.exe" -Destination "C:\nezha\nezha-agent.exe"
if ($file = "nezha-agent_windows_amd64.zip") {
    Move-Item -Path "C:\temp\nssm-2.24\win64\nssm.exe" -Destination "C:\nezha\nssm.exe"
}
else {
    Move-Item -Path "C:\temp\nssm-2.24\win32\nssm.exe" -Destination "C:\nezha\nssm.exe"
}
#清理垃圾
Remove-Item "C:\nezha.zip"
Remove-Item "C:\nssm.zip"
Remove-Item "C:\temp" -Recurse
#安装部分
C:\nezha\nssm.exe install nezha C:\nezha\nezha-agent.exe -s $server -p $key $tls -d 
C:\nezha\nssm.exe start nezha
#enjoy
Write-Host "Enjoy It!" -BackgroundColor DarkGreen -ForegroundColor Red
