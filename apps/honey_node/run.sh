#!/bin/bash
echo "Building honey_node..."
go build -o honey_node main.go
if [ $? -eq 0 ]; then
    echo "Build successful, running with sudo..."
    sudo ./honey_node
else
    echo "Build failed!"
fi
