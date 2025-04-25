# Dockerfile

# --- Stage 1: Build ---
# 使用 Node 镜像来构建 Vue 应用
FROM node:18-alpine as builder

WORKDIR /app

# 复制 package.json 和 lock 文件
COPY package*.json ./

# 安装依赖
RUN npm install

# 复制项目文件
COPY . .

# 运行构建命令 (Vite 会将 public/config.js 复制到 dist/)
RUN npm run build

# --- Stage 2: Production ---
# 使用 Nginx 镜像来提供服务
FROM nginx:1.25-alpine

# Nginx 配置 (可选，如果默认配置满足需求则不需要)
# COPY nginx.conf /etc/nginx/nginx.conf
# COPY default.conf /etc/nginx/conf.d/default.conf 

# 从构建阶段复制构建好的静态文件到 Nginx 的 www 目录
COPY --from=builder /app/dist /usr/share/nginx/html

# 复制入口点脚本到镜像中
COPY docker-entrypoint.sh /docker-entrypoint.sh

# 赋予脚本执行权限
RUN chmod +x /docker-entrypoint.sh

# 设置入口点脚本
ENTRYPOINT ["/docker-entrypoint.sh"]

# 默认 Nginx 会在前台运行，如果你的 entrypoint 脚本最后没有 exec nginx ...
# 你可能需要在这里指定 CMD
# CMD ["nginx", "-g", "daemon off;"] 
# 但由于 entrypoint 脚本最后执行了 exec nginx...，这里可以省略或注释掉