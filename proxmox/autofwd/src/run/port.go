package run

import (
	"autofwd/src/logger"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

const (
	PORT_START = 30001
	PORT_RANGE = 65535 - 30001
)

type GamePort struct {
	Minecraft string `yaml:"minecraft"`
}

type DNATConfig struct {
	Interface  string
	DestIP     string
	PublicPort int
}

type InterfacesInformation struct {
	Iface string
	IpNet *net.IPNet
	Ip    *net.IP
}

const GAMEPORT_YAML = "/var/lib/misc/gameport.yaml"

func parseYaml() (*GamePort, error) {
	gp := new(GamePort)
	temp := &bytes.Buffer{}

	f, err := os.Open(GAMEPORT_YAML)
	if err != nil {
		return nil, fmt.Errorf("error open file %s: %s", GAMEPORT_YAML, err)
	}
	if _, err = io.Copy(temp, f); err != nil {
		return nil, fmt.Errorf("error copy %s", err)
	}
	f.Close()
	buf := make([]byte, temp.Len())
	if _, err = temp.Read(buf); err != nil {
		return nil, fmt.Errorf("error read buf %s", err)
	}
	if err = yaml.Unmarshal(buf, &gp); err != nil {
		return nil, fmt.Errorf("error unmarshall %s", err)
	}
	return gp, nil
}

func creatreInterfacesInformation() []*InterfacesInformation {
	var infoIfaces []*InterfacesInformation

	interfaces, err := net.Interfaces()
	if err != nil {
		logger.Fatalf("error fetching interfaces: %s", err)
	}
	r, err := regexp.Compile(`.br[0-9]+`)
	if err != nil {
		logger.Fatalf("error compiling regexp: %s", err)
	}
	for _, iface := range interfaces {
		if !r.MatchString(iface.Name) {
			continue
		}
		if addrs, err := iface.Addrs(); err == nil {
			for _, addr := range addrs {
				if iip, ipNet, err := net.ParseCIDR(addr.String()); err == nil {
					if !iip.IsLoopback() && iip.IsPrivate() && iip.To4() != nil {
						infoIface := new(InterfacesInformation)
						infoIface.Iface = iface.Name
						infoIface.IpNet = ipNet
						infoIface.Ip = &iip
						infoIfaces = append(infoIfaces, infoIface)
					}
				}
			}
		}
	}
	return infoIfaces
}

// Use to find interface bridge that use the current ip.
func findingBridge(ip string) string {
	infoIfaces := creatreInterfacesInformation()
	for _, infoIface := range infoIfaces {
		if infoIface.IpNet.Contains(net.ParseIP(ip)) {
			return infoIface.Iface
		}
	}
	return ""
}

// Need to find bridge on proper interfaces.
func newDNATConfig(ip string) (*DNATConfig, error) {
	iface := findingBridge(ip)
	if iface == "" {
		return nil, fmt.Errorf("error no interface for: %s", ip)
	}
	cfg := new(DNATConfig)
	cfg.Interface = iface
	cfg.DestIP = ip
	return cfg, nil
}

// Forward port of gameserver to ip range of PORT_START to PORT_RANGE
func ForwardPort(ip, ports string) {
	cfg, err := newDNATConfig(ip)
	if err != nil {
		logger.Errf("%s", err)
		return
	}
	portValues := strings.Split(ports, ",")
	// need to port forward to PORT_START < PORT_RANGE,
	// so that we can use that port for different gameport or the same on 1 public ip.
	for _, port := range portValues {

	}
}

func TestingPort(ips []string) (map[int]bool, map[int][]string, error) {
	mapRet := make(map[int]bool)
	mapPort := make(map[int][]string)
	gp, err := parseYaml()
	if err != nil {
		return nil, nil, err
	}
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("error ips is empty")
	}
	r := reflect.ValueOf(gp).Elem()
	rt := r.Type()
	for i, ip := range ips {
		count := 0
		timeout := time.Second
		for val := 0; i < rt.NumField(); val++ {
			field := rt.Field(val)
			rv := reflect.ValueOf(gp)
			value := reflect.Indirect(rv).FieldByName(field.Name)
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, value.String()), timeout)
			if err != nil {
				logger.Errf("error failed to connect: %s", err)
				continue
			}
			if conn != nil {
				mapRet[i] = true
				ports := mapPort[i]
				ports = make([]string, rt.Len())
				ports[count] = value.String()
				mapPort[i] = ports
				count++
				logger.Printf("Opened %s", net.JoinHostPort(ip, value.String()))
				conn.Close()
			}
		}
	}
	return mapRet, mapPort, nil
}
