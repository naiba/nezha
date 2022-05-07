# User Guide

## Script for installation

**Recommended configuration：** Prepare _two domains_ before installation，a domain can **connect to CDN** for _Public Access_，for example (status.nai.ba). Another domain name resolves to the panel server allows the Agent can connect to the Dashboard，This domain **cannot connect to CDN** You need to make it expose the ip of the panel server directly.

```shell
curl -L https://raw.githubusercontent.com/naiba/nezha/master/script/install_en.sh  -o nezha.sh && chmod +x nezha.sh && sudo ./nezha.sh
```

_\* Use WatchTower to automatically update the panel, and in Windows you can use nssm to configure self-start_

**Windows One-Click Installation Agent (please use Powershell administrator privileges)**

```powershell
set-ExecutionPolicy RemoteSigned;Invoke-WebRequest https://raw.githubusercontent.com/naiba/nezha/master/script/install.ps1 -OutFile C:\install.ps1;powershell.exe C:\install.ps1 dashboard_host:grpc_port secret
```

_If you encounter the prompt "Implement Policy Change" please select Y_

### Customize Agent 

#### Customize the NIC and hard drive partitions to be monitored

Execute `/opt/nezha/agent/nezha-agent --edit-agent-config` to select a custom NIC and partition, and then restart Agent

#### Operating parameters

Execute `./nezha-agent --help` to view supported parameters，if you are already using the one-click script, you can edit `/etc/systemd/system/nezha-agent.service`，at the end of this line `ExecStart=` add:

- `--report-delay` System information reporting interval, default is 1 second, can be set to 3 to reduce the system resource usage on the agent side (configuration range 1-4)
- `--skip-conn` Not monitoring the number of connections, if it is a server with a large number of connections, the CPU usage will be high. It is recommended to set this to reduce CPU usage
- `--skip-procs` Disable monitoring the number of processes can also reduce CPU and memory usage
- `--disable-auto-update` Disable **Automatic Update** Agent (security feature)
- `--disable-force-update` Disable **Forced Update** Agent (security feature)
- `--disable-command-execute` Disable execution of scheduled tasks, disallow open online terminals on the Agent side (security feature)
- `--tls` Enable SSL/TLS encryption (If you are using nginx to reverse proxy Agent´s grpc connections, and if nginx has SSL/TLS enabled, you need to enable this configuration)

## Description of the functions

<details>
    <summary>Scheduled tasks: backup scripts, service restarts, and other scheduled tasks</summary>

Use this feature to periodically back up the server in combination with restic or rclone, or to periodically restart a service to reset the network connection.

</details>

<details>
    <summary>Notification: Real-time monitoring of load, CPU, memory, hard disk, bandwidth, transfer, monthly transfer, number of processes, number of connections</summary>

#### Flexible notification methods

`#NEZHA#` is a panel message placeholder, and the panel will automatically replace the placeholder with the actual message when it triggers a notification

The content of Body is  in `JSON` format：**When the request type is FORM**，the value is in the form of `key:value`，`value` can contain placeholders that will be automatically replaced when notified. **When the request type is JSON** It will only do string substitution and submit to the `URL` directly.

Placeholders can also be placed inside the URL, and it will perform a simple string substitution when requested.

Refer to the example below, it is very flexible.

1. Add notification method

   - Telegram Example, contributed by [@haitau](https://github.com/haitau)

     - Name：Telegram Robot message notification
     - URL：<https://api.telegram.org/botXXXXXX/sendMessage?chat_id=YYYYYY&text=#NEZHA>#
     - Request method: GET
     - Request type: default
     - Body: null
     - URL Parameter acquisition instructions：The XXXXXX in botXXXXXX is the token provided when you follow the official @Botfather in Telegram and enter /newbot to create a new bot. (In the line after _Use this token to access the HTTP API_). The 'bot' are essential. After creating a bot, you need to talk to the BOT in Telegram (send a random message) before you can send a message by using API. YYYYYY is Telegram user's ID, you can get it by talking to the bot @userinfobot.

2. Add an offline notification

   - Name: Offline notifications
   - Rule: `[{"Type":"offline","Duration":10}]`
   - Enable: √

3. Add an notification when the CPU exceeds 50% for 10s **but** the memory usage is below 20% for 20s

   - Name: CPU+RAM
   - Rule: `[{"Type":"cpu","Min":0,"Max":50,"Duration":10},{"Type":"memory","Min":20,"Max":0,"Duration":20}]`
   - Enable: √

#### Description of notification rules

##### Basic Rules

- Type
  - `cpu`、`memory`、`swap`、`disk`
  - `net_in_speed` Inbound speed, `net_out_speed` Outbound speed, `net_all_speed` Inbound + Outbound speed, `transfer_in` Inbound Transfer, `transfer_out` Outbound Transfer, `transfer_all` Total Transfer
  - `offline` Offline monitoring
  - `load1`、`load5`、`load15` load
  - `process_count` Number of processes _Currently, counting the number of processes takes up too many resources and is not supported at the moment_
  - `tcp_conn_count`、`udp_conn_count` Number of connections
- duration：Lasting for a few seconds, the notification will only be triggered when the sampling record reaches 30% or more within a few seconds
- min/max
  - Transfer, network speed, and other values of the same type. Unit is byte (1KB=1024B，1MB = 1024\*1024B)
  - Memory, hard disk, CPU. units are usage percentages
  - No setup required for offline monitoring
- cover `[{"type":"offline","duration":10, "cover":0, "ignore":{"5": true}}]`
  - `0` Cover all, use `ignore` to ignore specific servers
  - `1` Ignore all, use `ignore` to monitoring specific servers
- ignore: `{"1": true, "2":false}` to ignore specific servers, use with `cover`

##### Special: Any-cycle transfer notification

Can be used as monthly transfer notificatin

- type
  - transfer_in_cycle Inbound transfer during the cycle
  - transfer_out_cycle Outbound transfer during the cycle
  - transfer_all_cycle The sum of inbound and outbound transfer during the cycle
- cycle_start Start date of the statistical cycle (can be the start date of your server's billing cycle), the time format is RFC3339, for example, the format in Beijing time zone is`2022-01-11T08:00:00.00+08:00`
- cycle_interval Interval time cycle  (For example, if the cycle is in days and the value is 7, it means that the statistics are counted every 7 days)
- cycle_unit Statistics cycle unit, default `hour`, optional(`hour`, `day`, `week`, `month`, `year`)
- min/max、cover、ignore Please refer to the basic rules to configure
- Example: The server with ID 3 (defined in the `ignore`) is counted on the 15th of each month, and a notification is triggered when the monthly outbound traffic reaches 1TB during the cycle. `[{"type":"transfer_out_cycle","max":1000000000000,"cycle_start":"2022-01-11T08:00:00.00+08:00","cycle_interval":1,"cycle_unit":"month","cover":1,"ignore":{"3":true}}]`
  ![7QKaUx.md.png](https://s4.ax1x.com/2022/01/13/7QKaUx.md.png)

</details>

<details>
    <summary>Service monitoring: HTTP, SSL certificate, ping, TCP port, etc.</summary>

Just go to the `/service` page and click on Add Service Monitor, there are instructions on the form.

</details>

<details>
  <summary>Custom code: change logo, change color tone, add statistics code, etc.</summary>

**Effective only on the visitor's home page.**

- Example of changing the default theme progress bar color

  ```html
  <style>
  .ui.fine.progress> .bar {
      background-color: pink !important;
  }
  </style>
  ```

- Example of modifying DayNight theme progress bar color and footer (by [@hyt-allen-xu](https://github.com/hyt-allen-xu))

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

- Example of modifying the logo of the default theme, modifying the footer (by [@iLay1678](https://github.com/iLay1678))

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
  footer.innerHTML="Powered by YOUR NAME"
  footer.style.visibility="visible"
  avatar.src="Your square logo link"
  avatar.style.visibility="visible"
  }
  </script>
  ```

- Example of modifying the background image of hotaru theme

  ```html
  <style>
  .hotaru-cover {
     background: url(https://s3.ax1x.com/2020/12/08/DzHv6A.jpg) center;
  }
  </style>
  ```

</details>

## FAQ

<details>
    <summary>How do I migrate my data to the new server and restore my backups?</summary>

1. First use the one-click script and select `Stop Panel`
2. Compress the `/opt/nezha` folder to the same path as the new server
3. Using the one-click script, select `Launch Panel`

</details>

<details>
    <summary>Let the Agent start/on-line, and the self-test process of the problem</summary>

1. Execute `/opt/nezha/agent/nezha-agent -s IP/Domin(Panel IP or Domain not connected to CDN):port(Panel RPC port) -p secret(Agent Secret) -d` Check the logs to see if the timeout is due to a DNS problem or poor network
2. `nc -v domain/IP port(Panel RPC port)` or `telnet domain/IP port(Panel RPC port)` to check if it' s a network problem, check the inbound and outbound firewall between the local machine and the panel server, if you can' t determine the problem you can check it with the port checking tool provided by <https://port.ping.pe/>.
3. If the above steps work and the Agent is online, please try to turn off SELinux on the panel server. [How to close SELinux？](https://www.google.com/search?q=How+to+close+SELinux)

</details>

<details>
    <summary>How to make the old version of OpenWRT/LEDE self-boot?</summary>

Refer to this project: <https://github.com/Erope/openwrt_nezha>

</details>

<details>
    <summary>How to make the new version of OpenWRT self-boot? By @艾斯德斯</summary>

First download the corresponding binary from the release, unzip the zip package and place it in `/root`, then execute `chmod +x /root/nezha-agent` to give it execute access, create file `/etc/init.d/nezha-service`：

```shell
#!/bin/sh /etc/rc.common

START=99
USE_PROCD=1

start_service() {
 procd_open_instance
 procd_set_param command /root/nezha-agent -s Domin/IP:port -p screat -d
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

Give it permission to execute `chmod +x /etc/init.d/nezha-service` then start the service `/etc/init.d/nezha-service enable && /etc/init.d/nezha-service start`

</details>

<details>
    <summary>Real-time channel disconnection/online terminal connection failure</summary>

Using a reverse proxy requires special configuration of the WebSocket for the `/ws` and `/terminal` paths to support real-time server status updates and **WebSSH**

- Nginx(Aapanel)：Add the following code to your nginx configuration file

  ```nginx
  server{

      #Some original configurations
      #server_name blablabla...

      location ~ ^/(ws|terminal/.+)$  {
          proxy_pass http://ip:site access port;
          proxy_set_header Upgrade $http_upgrade;
          proxy_set_header Connection "Upgrade";
          proxy_set_header Host $host;
      }

      #Others, such as location blablabla...
  }
  ```

  If you're not using Aapanel, add this code to the `server{}`:

  ```nginx
  location / {
    proxy_pass http://ip:port(Access port);
    proxy_set_header Host $host;
  }
  ```

- CaddyServer v1（v2 no special configuration required）

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
    <summary>Reverse Proxy gRPC Port (support Cloudflare CDN)</summary>
Use Nginx or Caddy to reverse proxy gRPC

- Nginx configuration files

```nginx
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ip-to-dashboard.nai.ba; # The domain name where the Agent connects to Dashboard

    ssl_certificate          /data/letsencrypt/fullchain.pem; # Your domain certificate path
    ssl_certificate_key      /data/letsencrypt/key.pem;       # Your domain's private key path

    underscores_in_headers on;

    location / {
        grpc_read_timeout 300s;
        grpc_send_timeout 300s;
        grpc_pass grpc://localhost:5555;
    }
}
```

- Caddy configuration files

```Caddyfile
ip-to-dashboard.nai.ba:443 { # The domain name where the Agent connects to Dashboard
    reverse_proxy {
        to localhost:5555
        transport http {
            versions h2c 2
        }
    }
}
```

Dashboard Panel Configuration

- First login to the panel and enter the admin panel, go to the settings page, fill in the `CDN Bypassed Domain/IP` with the domain name you configured in Nginx or Caddy, for example `ip-to-dashboard.nai.ba`, and save it.
- Then open the /opt/nezha/dashboard/data/config.yaml file in the panel server and change `proxygrpcport` to the port that Nginx or Caddy is listening on, such as `443` as set in the previous step. Since we have SSL/TLS enabled in Nginx or Caddy, we need to set `tls` to `true`, restart the panel when you are done.

Agent Configuration

- Log in to the admin panel, copy the one-click install command, and execute the one-click install command on the corresponding server to reinstall the agent.

Enable Cloudflare CDN (optional)

According to Cloudflare gRPC requirements: gRPC services must listen on port 443 and must support TLS and HTTP/2.
So if you need to enable CDN, you must use port 443 when configuring Nginx or Caddy reverse proxy gRPC and configure the certificate (Caddy will automatically apply and configure the certificate).

-  Log in to Cloudflare and select the domain you are using. Go to the `Network` page and turn on the `gRPC` switch, then go to the `DNS` page, find the resolution record of the domain with gRPC configuration, and turn on the orange cloud icon to enable CDN.

</details>
