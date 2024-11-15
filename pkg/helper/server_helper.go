//服务器帮助类

package helper

import (
	"net"
	"reflect"
	"strings"

	"github.com/oklog/ulid/v2"
)

// GetServerIp
// 问题：我在本地多网卡机器上，运行分布式场景，此函数返回的ip有误导致rpc连接失败。 遂google结果如下：
// 1、https://www.jianshu.com/p/301aabc06972
// 2、https://www.cnblogs.com/chaselogs/p/11301940.html
func GetServerIp() string {
	ip, err := externalIP()
	if err != nil {
		return ""
	}
	return ip.String()
}

// GetLastSegmentOfIP 返回给定IP地址的最后一段数字。
func GetLastSegmentOfIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func externalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, err
}

func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil // not an ipv4 address
	}

	return ip
}

// 生成唯一id
func CreateUniqueId() (uuid string) {

	randUuid := ulid.Make()

	uuid = randUuid.String()
	return
}

// 将map转换为结构体
func MapToStruct(m map[string]interface{}, s interface{}) error {
	// 获取结构体的值和类型
	val := reflect.ValueOf(s).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := typ.Field(i).Name

		// 从 map 中获取对应的值
		if value, ok := m[fieldName]; ok {
			// 使用反射设置结构体字段的值
			if field.CanSet() {
				fieldValue := reflect.ValueOf(value)
				if fieldValue.Type().ConvertibleTo(field.Type()) {
					field.Set(fieldValue.Convert(field.Type()))
				}
			}
		}
	}
	return nil
}
