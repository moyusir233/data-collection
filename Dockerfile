FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
    && rm -rf /var/lib/apt/lists/ \
    && apt-get autoremove -y && apt-get autoclean -y \

EXPOSE 8000
EXPOSE 9000

# 存放编译好后的二进制可执行文件的目录
VOLUME /app
# 存放配置文件的目录
VOLUME /etc/app-configs

WORKDIR /app

CMD ["./server", "-conf", "/etc/app-configs/config.yaml"]
