#!/bin/bash
echo "Building honey_server..."
go build -o honey_server main.go
if [ $? -eq 0 ]; then
    echo "Build successful, running with sudo..."
    sudo ./honey_server
else
    echo "Build failed!"
fi
