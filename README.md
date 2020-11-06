# 哪吒面板

阶段： *alpha*

服务期状态监控，被动接收（非 node-exporter 那种主动拉取的方式。）

## 系统设计

C/S 采用 gRPC 通信，客户端通过添加主机生成的单独 Token 上报监控信息。因为不会做成多用户的，上报信息会储存到内存中，暂不提供历史数据统计。

- 首次连接：上报基本信息（系统、CPU基本信息），后面管理员可从客户端主动拉取更新。
- 监控上报：每隔 3s 向服务器上报系统信息

## 部署指南

### 控制面板 + 节点一键启动

1. 创建一个文件夹

    ```shell
    mkdir nezha
    ```

2. 进入文件夹并创建 `docker-compose.yaml` 文件

    ```shell
    cd nezha && nano docker-compose.yaml
    ```

    将以下内容粘贴进去，注意查看 `environment` 中的几项配置。ID、密钥是在管理面板添加服务器之后才有的，不是你的 GitHub ID。
  
    ```yaml
    version: "3.3"

    services:
      dashboard:
        image: docker.pkg.github.com/p14yground/nezha/dashboard
        restart: always
        volumes:
          - ./data:/dashboard/data
        ports:
          - 8008:80
          - 5555:5555
      agent:
        image: docker.pkg.github.com/p14yground/nezha/agent
        depends_on:
          - dashboard
        environment:
          - ID=1 #节点ID，启动后在管理后台添加后显示
          - SECRET=secret #节点密钥，启动后在管理后台添加后显示
          - SERVER=ops.naibahq.com:5555 #服务器RPC端口
          - DEBUG=false #服务器地址使用IP时设置为true
        volumes:
          - /proc:/agent/host/proc:ro
          - /sys:/agent/host/sys:ro
          - /etc:/agent/host/etc:ro
          - /var:/agent/host/var:ro
          - /run:/agent/host/run:ro
          - /dev:/agent/host/dev:ro
    ```

3. 创建控制面板配置文件

    ```shell
    mkdir data && nano config.yaml
    ```

    将以下内容粘贴进去

    ```yaml
    debug: true
    httpport: 80
    github:
      admin: # 多管理员
        - 1 #管理员 GitHub ID，复制自己GitHub头像图片地址，/[ID].png
        - 2
      clientid: GitHub Oauth App clientID # 在 https://github.com/settings/developers 创建，无需审核 Callback 填 http(s)://域名或IP/oauth2/callback
      clientsecret: client secret
    site:
      brand: 站点标题
      cookiename: tulong #浏览器 Cookie 字段名，可不改
    ```

4. 启动管理面板

    ```shell
    docker-compose up -d
    ```

5. 更新，可以使用下方的命令，或者配置 **WatchTower** 自动更新所有容器

    ```shell
    docker-compose pull && docker-compose up -d
    ```

6. *agent* 配置：登入你的管理面板添加服务器，把节点的 ID、密钥 编辑进 `docker-compose.yaml` 文件中，然后重启 agent。

    ```shell
    docker-compose restart agent
    ```

### 单节点部署

1. 登入你的管理面板添加服务器，把节点的 ID、密钥 记录下来，下面会用到。

2. 创建一个文件夹

    ```shell
    mkdir nezha
    ```

3. 进入文件夹并创建 `docker-compose.yaml` 文件，将 ID、密钥 编辑进去。

    ```shell
    cd nezha && nano docker-compose.yaml
    ```

    将以下内容粘贴进去，ID、密钥是在管理面板添加服务器之后才有的，不是你的 GitHub ID。
  
    ```yaml
    version: "3.3"

    services:
      agent:
        image: docker.pkg.github.com/p14yground/nezha/agent
        environment:
          - ID=1 #节点ID，启动后在管理后台添加后显示
          - SECRET=secret #节点密钥，启动后在管理后台添加后显示
          - SERVER=ops.naibahq.com:5555 #服务器RPC端口
          - DEBUG=false #服务器地址使用IP时设置为true
        volumes:
          - /proc:/agent/host/proc:ro
          - /sys:/agent/host/sys:ro
          - /etc:/agent/host/etc:ro
          - /var:/agent/host/var:ro
          - /run:/agent/host/run:ro
          - /dev:/agent/host/dev:ro
    ```

4. 启动

    ```shell
    docker-compose up -d
    ```

5. 更新，可以使用下方的命令，或者配置 **WatchTower** 自动更新所有容器

    ```shell
    docker-compose pull && docker-compose up -d
    ```

**Windows、MacOS、Andorid 等也可监控，需要参照教程文章里面的文章编译 agent 并启动。**

## 教程文章

 - [哪吒面板：小鸡们的最佳探针](https://www.zhujizixun.com/2843.html)*（已过时）*