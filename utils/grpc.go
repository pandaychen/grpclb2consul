package utils

import (
	//"string"
	"fmt"
	"net"
)

func GetGrpcScheme(scheme string) string {
	return fmt.Sprintf("%s:///", scheme)
}

// GetOutboundIP 获取本机的出口IP
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}
