package mq_service

import (
	"fmt"
)

func BindPortExChange(msg string) error {
	fmt.Println("端口绑定消息", msg)
	return nil
}
