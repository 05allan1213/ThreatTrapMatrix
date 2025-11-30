package mq_service

import (
	"fmt"
)

func DeleteIpExChange(msg string) error {
	fmt.Println("删除消息", msg)
	return nil
}
