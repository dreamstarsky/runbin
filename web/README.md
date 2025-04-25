# How to run?
1. 直接启动
- 修改 `public/config.js` 里的地址
- 运行 `npm run dev`

2. 启动docker
```bash
docker run -d --name=web-server -p 80:80 -e BACKEND_URL=114514 -e LSP_SERVER_URL=1919810 hxzzz/meow-paste:latest
```

3. 语言服务器
```bash
docker run -d --name=lsp-server -p 3003:3003 hxzzz/cpplsp:latest
```
cpp 语言服务器
`ws://127.0.0.1:3003/cpp`