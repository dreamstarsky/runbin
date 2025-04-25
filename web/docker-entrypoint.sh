#!/bin/sh
# docker-entrypoint.sh

# 从环境变量读取配置，如果未设置，则使用占位符作为默认值（或设置一个开发默认值）
# 我们将使用 BACKEND_URL 和 LSP_SERVER_URL 作为环境变量的名称
RUNTIME_BACKEND_URL=${BACKEND_URL:-"__BACKEND_URL_PLACEHOLDER__"}
RUNTIME_LSP_SERVER_URL=${LSP_SERVER_URL:-"__LSP_SERVER_URL_PLACEHOLDER__"}

echo "Runtime Backend URL from env: ${RUNTIME_BACKEND_URL}"
echo "Runtime LSP Server URL from env: ${RUNTIME_LSP_SERVER_URL}"

# 目标配置文件路径 (根据你的 Nginx 配置调整)
CONFIG_FILE_PATH="/usr/share/nginx/html/config.js"
BACKEND_PLACEHOLDER="__BACKEND_URL_PLACEHOLDER__"
LSP_PLACEHOLDER="__LSP_SERVER_URL_PLACEHOLDER__"

# 检查文件是否存在
if [ -f "$CONFIG_FILE_PATH" ]; then
  echo "Replacing placeholders in $CONFIG_FILE_PATH"
  # 使用 sed 进行替换。使用 # 作为分隔符，以避免 URL 中的 / 导致的问题
  # 替换 BACKEND URL
  sed -i "s#${BACKEND_PLACEHOLDER}#${RUNTIME_BACKEND_URL}#g" "$CONFIG_FILE_PATH"
  # 替换 LSP SERVER URL
  sed -i "s#${LSP_PLACEHOLDER}#${RUNTIME_LSP_SERVER_URL}#g" "$CONFIG_FILE_PATH"
  # (可选) 修改初始化标记
  sed -i "s/window.CONFIG_INITIALIZED_BY_ENTRYPOINT = false;/window.CONFIG_INITIALIZED_BY_ENTRYPOINT = true;/g" "$CONFIG_FILE_PATH"
  echo "Placeholders replaced."
else
  echo "Error: Config file $CONFIG_FILE_PATH not found!"
  exit 1
fi

# 执行 Docker 镜像的默认命令 (通常是启动 Nginx)
echo "Starting Nginx..."
exec nginx -g 'daemon off;'