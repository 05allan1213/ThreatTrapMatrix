package mq_service

import "fmt"

func CreateIpExChange(msg string) error {
	fmt.Println("消息", msg)
	return nil
}
