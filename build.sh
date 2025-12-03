#!/bin/bash

# 1. 构建前端
echo "[*] Building frontend..."
cd frontend
npm install
npm run build
cd ..

# 2. 构建 Go 后端
#    -ldflags "-w -s" 用于减小二进制文件大小
echo "[*] Building backend..."
go build -ldflags "-w -s" -o subsonic main.go

echo "[+] Build complete! Run ./subsonic"
