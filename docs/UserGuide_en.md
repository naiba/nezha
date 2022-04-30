# User Guide

## install script

**Recommended configuration：** Preparation before installation _Two domains_，one can **access CDN** as _Public Access_，for example (status.nai.ba)；Another one resolves to the panel server as Agent connect Dashboard use，**can't access CDN** Direct exposure panel host IP，for example（ip-to-dashboard.nai.ba）。

```shell
curl -L https://raw.githubusercontent.com/naiba/nezha/master/script/install_en.sh  -o nezha.sh && chmod +x nezha.sh
sudo ./nezha.sh
```

_\* use WatchTower Panels can be updated automatically，Windows terminal can use nssm configure autostart_

**Windows -A key installation Agent （please use Powershell admin rights）**

```powershell
set-ExecutionPolicy RemoteSigned;Invoke-WebRequest https://raw.githubusercontent.com/naiba/nezha/master/script/install.ps1 -OutFile C:\install.ps1;powershell.exe C:\install.ps1 dashboard_host:grpc_port secret
```

_In case of confirmation「Implement policy changes」please choose Y_

### Agent customize

#### Custom monitoring of network cards and hard disk partitions

implement `/opt/nezha/agent/nezha-agent --edit-agent-config` to select custom NICs and partitions，then reboot just agent

#### Operating parameters

by executing `./nezha-agent --help` View supported parameters，If you use one-click scripting，can be edited `/etc/systemd/system/nezha-agent.service`，exist `ExecStart=` At the end of this line add

- `--report-delay` System information reporting interval，The default is 1 Second，can be set to 3 to further reduce agent End-system resource usage（Configuration interval 1-4）
- `--skip-conn` Do not monitor the number of connections，if vpn-gateway/connection-intensive machines High CPU usage，Recommended settings
- `--skip-procs` Do not monitor the number of processes，can also be reduced agent occupy
- `--disable-auto-update` prohibit **auto update** Agent（safety features）
- `--disable-force-update` prohibit **Force update** Agent（safety features）
- `--disable-command-execute` prohibited in Agent Execute scheduled tasks on the machine、Open online terminal（safety features）
- `--tls` enable SSL/TLS encryption（use nginx reverse proxy Agent of grpc connect，and nginx turn on SSL/TLS Time，This configuration needs to be enabled）

## Function Description

<details>
    <summary>Scheduled Tasks：backup script、service restart，and other periodic operation and maintenance tasks。</summary>

Use this feature to periodically combine restic、rclone back up the server，Or periodically restart some service to reset the network connection。

</details>

<details>
    <summary>Alarm notification：Real-time monitoring of load, CPU, memory, hard disk, bandwidth, traffic, monthly traffic, number of processes, and number of connections。</summary>

#### Flexible notification methods

`#NEZHA#` is the panel message placeholder，The panel will automatically replace the placeholder with the actual message when the notification is triggered

Body content is`JSON` formatted：**when the request type is FORM Time**，value is `key:value` form，`value` Placeholders can be placed inside，Automatically replace when notified。**when the request type is JSON** It will only be submitted directly to the`URL`。

URL Placeholders can also be placed inside，Simple string replacement is done when requested。

Refer to the example below，very flexible。

1. Add notification method

   - telegram Example [@haitau](https://github.com/haitau) contribute

     - name：telegram Robot message notification
     - URL：<https://api.telegram.org/botXXXXXX/sendMessage?chat_id=YYYYYY&text=#NEZHA>#
     - request method: GET
     - request type: default
     - Body: null
     - URL Parameter acquisition instructions：botXXXXXX Neutral XXXXXX is in telegram Follow the official @Botfather ，enter/newbot ，Create new bot（bot）Time，will provide token（in prompt Use this token to access the HTTP API:next line）here 'bot' Three letters are indispensable. After bot created, You need to chat with the BOT to have a conversation（Just send a message），then available API Send a message. YYYYYY is telegram user's number ID。with the robot @userinfobot Dialogue is available。

2. Add an offline alarm

   - name：Offline notifications
   - rule：`[{"Type":"offline","Duration":10}]`
   - enable：√

3. add a monitor CPU continued 10s Exceed 50% **and** memory persistent 20s Occupied less than 20% the alarm

   - name：CPU+RAM
   - rule：`[{"Type":"cpu","Min":0,"Max":50,"Duration":10},{"Type":"memory","Min":20,"Max":0,"Duration":20}]`
   - enable：√

#### Description of alarm rules

##### basic rules

- type
  - `cpu`、`memory`、`swap`、`disk`
  - `net_in_speed` Inbound speed、`net_out_speed` Outbound speed、`net_all_speed` two-way speed、`transfer_in` Inbound traffic、`transfer_out` Outbound traffic、`transfer_all` bidirectional traffic
  - `offline` Offline monitoring
  - `load1`、`load5`、`load15` load
  - `process_count` number of processes _Currently fetching threads takes up too many resources，Temporarily not supported_
  - `tcp_conn_count`、`udp_conn_count` number of connections
- duration：duration in seconds，Sampling records in seconds 30% The above trigger threshold will only alarm（Anti-Data Pin）
- min/max
  - flow、Network speed class value as bytes（1KB=1024B，1MB = 1024\*1024B）
  - memory、hard disk、CPU occupancy percentage
  - Offline monitoring without setup
- cover `[{"type":"offline","duration":10, "cover":0, "ignore":{"5": true}}]`
  - `0` monitor all，pass `ignore` ignore specific server
  - `1` ignore all，pass `ignore` Monitor specific servers
- ignore: `{"1": true, "2":false}` specific server，match `cover` use

##### special：Arbitrary cycle flow alarm

Can be used as monthly flow alarm

- type
  - transfer_in_cycle Inbound traffic during the period
  - transfer_out_cycle Outbound traffic during the period
  - transfer_all_cycle Bidirectional flow in cycles and
- cycle_start Fiscal Period Start Date（Can be the start date of your machine billing cycle），RFC3339 Time format，For example, Beijing time is`2022-01-11T08:00:00.00+08:00`
- cycle_interval How many cycle units every (for example, if the cycle unit is days, the value is 7, which means that the statistics will be counted every 7 days）
- cycle_unit Statistical period unit, default `hour`, optional(`hour`, `day`, `week`, `month`, `year`)
- min/max、cover、ignore Refer to Basic Rules Configuration
- Example: ID for 3 the machine（ignore inside the definition）of monthly 15 outbound monthly traffic billed 1T Call the police `[{"type":"transfer_out_cycle","max":1000000000000,"cycle_start":"2022-01-11T08:00:00.00+08:00","cycle_interval":1,"cycle_unit":"month","cover":1,"ignore":{"3":true}}]`
  ![7QKaUx.md.png](https://s4.ax1x.com/2022/01/13/7QKaUx.md.png)

</details>

<details>
    <summary>service monitoring：HTTP、SSL certificate、ping、TCP port etc。</summary>

Enter `/monitor` Click to create a new monitor on the page，Instructions are below the form。

</details>

<details>
  <summary>custom code：Change the logo、change color、Add statistical code, etc.。</summary>

**Effective only on the visitor's home page.**

- Default theme changing progress bar color example

  ```html
  <style>
  .ui.fine.progress> .bar {
      background-color: pink !important;
  }
  </style>
  ```

- DayNight Example of theme changing progress bar color, modifying footer（from [@hyt-allen-xu](https://github.com/hyt-allen-xu)）

  ```html
  <style>
  .ui.fine.progress> .progress-bar {
    background-color: #00a7d0 !important;
  }
  </style>
  <script>
  window.onload = function(){
  var footer=document.querySelector("div.footer-container")
  footer.innerHTML="©2021 "your name" & Powered by "your name"
  footer.style.visibility="visible"
  }
  </script>
  ```

- Default theme modification LOGO、Modify footer example（from [@iLay1678](https://github.com/iLay1678)）

  ```html
  <style>
  .right.menu>a{
  visibility: hidden;
  }
  .footer .is-size-7{
  visibility: hidden;
  }
  .item img{
  visibility: hidden;
  }
  </style>
  <script>
  window.onload = function(){
  var avatar=document.querySelector(".item img")
  var footer=document.querySelector("div.is-size-7")
  footer.innerHTML="Powered by your name"
  footer.style.visibility="visible"
  avatar.src="your square logo address"
  avatar.style.visibility="visible"
  }
  </script>
  ```

- hotaru Theme change background image example

  ```html
  <style>
  .hotaru-cover {
     background: url(https://s3.ax1x.com/2020/12/08/DzHv6A.jpg) center;
  }
  </style>
  ```

</details>

## common problem

<details>
    <summary>How to perform data migration、Backup and restore？</summary>

1. First use one-click script `stop panel`
2. Pack `/opt/nezha` folder, to the same location in the new environment
3. Use one-click script `Launchpad`

</details>

<details>
    <summary>Agent Start/Go Online Problem Self-Check Process</summary>

1. direct execution `/opt/nezha/agent/nezha-agent -s Panel IP or non-CDN domain name:Panel RPC port -p Agent key -d` Check if the log is DNS、Poor network causes timeout（timeout） question。
2. `nc -v domain name/IP Panel RPC port` or `telnet domain name/IP Panel RPC port` Check if it is a network problem，Check local and panel server inbound and outbound firewalls，If the single machine cannot judge, you can use the <https://port.ping.pe/> Provided port inspection tool for detection。
3. If the above steps detect normal，Agent normal online，try to close SELinux，[how to close SELinux？](https://www.google.com/search?q=%E5%85%B3%E9%97%ADSELINUX)

</details>

<details>
    <summary>how to make Legacy OpenWRT/LEDE self-start？</summary>

refer to this project: <https://github.com/Erope/openwrt_nezha>

</details>

<details>
    <summary>how to make New version of OpenWRT self-start？via @esdes</summary>

first in release Download the corresponding binary decompression zip After the package is placed in `/root`，Then `chmod +x /root/nezha-agent` give execute permission，then create `/etc/init.d/nezha-service`：

```shell
#!/bin/sh /etc/rc.common

START=99
USE_PROCD=1

start_service() {
 procd_open_instance
 procd_set_param command /root/nezha-agent -s Panel URL:receive port -p unique key -d
 procd_set_param respawn
 procd_close_instance
}

stop_service() {
    killall nezha-agent
}

restart() {
 stop
 sleep 2
 start
}
```

give execute permission `chmod +x /etc/init.d/nezha-service` then start the service `/etc/init.d/nezha-service enable && /etc/init.d/nezha-service start`

</details>

<details>
    <summary>Live channel disconnected/Online terminal connection failed</summary>

When using a reverse proxy, you need to target `/ws`,`/terminal` path WebSocket Specially configured to support real-time server status updates and **WebSSH**。

- Nginx(Aapanel/Pagoda)：At your nginx Add the following code to the configuration file

  ```nginx
  server{

      #some original configuration
      #server_name blablabla...

      location ~ ^/(ws|terminal/.+)$  {
          proxy_pass http://ip:site access port;
          proxy_set_header Upgrade $http_upgrade;
          proxy_set_header Connection "Upgrade";
          proxy_set_header Host $host;
      }

      #others location blablabla...
  }
  ```

  If not a Aapanel/Pagoda, still in `server{}` add this paragraph

  ```nginx
  location / {
    proxy_pass http://ip:site access port;
    proxy_set_header Host $host;
  }
  ```

- CaddyServer v1（v2 No special configuration required）

  ```Caddyfile
  proxy /ws http://ip:8008 {
      websocket
  }
  proxy /terminal/* http://ip:8008 {
      websocket
  }
  ```

</details>

<details>
    <summary>reverse proxy gRPC port（support Cloudflare CDN）</summary>
use Nginx or Caddy reverse proxy gRPC

- Nginx configure

```nginx
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ip-to-dashboard.nai.ba; # yours Agent connect Dashboard's domain name

    ssl_certificate          /data/letsencrypt/fullchain.pem; # your domain certificate path
    ssl_certificate_key      /data/letsencrypt/key.pem;       # Your domain name private key path

    underscores_in_headers on;

    location / {
        grpc_read_timeout 300s;
        grpc_send_timeout 300s;
        grpc_pass grpc://localhost:5555;
    }
}
```

- Caddy configure

```Caddyfile
ip-to-dashboard.nai.ba:443 { # yours Agent connect Dashboard's domain name
    reverse_proxy {
        to localhost:5555
        transport http {
            versions h2c 2
        }
    }
}
```

Dashboard Panel side configuration

- First log in to the panel to enter the management background Open the settings page，exist `Panel server domain name that is not connected to CDN/IP` Fill in the previous step in Nginx or Caddy domain name configured in for example `ip-to-dashboard.nai.ba` ，and save。
- then in the panel server，Open /opt/nezha/dashboard/data/config.yaml 文件，将 `proxygrpcport` change into Nginx or Caddy listening port，or as set in the previous step `443` ；because we are Nginx or Caddy turned on SSL/TLS，So it is necessary to `tls` Set as `true` ；Restart the panel after modification is complete。

Agent end configuration

- Login panel management background，Copy the one-click install command，Execute the one-click installation command on the corresponding server to reinstall agent end。

turn on Cloudflare CDN（optional）

according to Cloudflare gRPC requirements：gRPC Service must listen 443 port and must support TLS and HTTP/2。
So if you need to turn it on CDN，must be configured Nginx or Caddy reverse proxy gRPC use when 443 port，and configure the certificate（Caddy Will automatically apply and configure the certificate）。

- Log in Cloudflare，Choose a domain name to use。Open `The internet` option will `gRPC` switch on，Open `DNS` options，turn up Nginx or Caddy Anti-generation gRPC The resolution record of the configured domain name，Open Orange Cloud Enable CDN。

</details>
