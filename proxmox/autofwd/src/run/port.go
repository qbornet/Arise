package run

import (
	"autofwd/src/logger"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
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

var (
	portStart int          = PORT_START
	usedPort  map[int]bool = make(map[int]bool)
)

type GamePort struct {
	Minecraft string `yaml:"minecraft"`
}

type GameConfig struct {
	Gameport GamePort `yaml:"gameport"`
}

type DNATConfig struct {
	Interface           string
	SubNetworkInterface string
	DestIP              string
	PublicPort          int
}

type InterfacesInformation struct {
	Iface string
	IpNet *net.IPNet
	Ip    *net.IP
}

const GAMEPORT_YAML = "/var/lib/misc/gameport.yaml"

func parseYaml() (*GameConfig, error) {
	gp := new(GameConfig)
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
	if err = yaml.Unmarshal(buf, gp); err != nil {
		return nil, fmt.Errorf("error unmarshall %s", err)
	}
	return gp, nil
}

func creatreInterfacesInformation(ip string) []*InterfacesInformation {
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
					if !iip.IsLoopback() && iip.IsPrivate() && ipNet.Contains(net.ParseIP(ip)) && iip.To4() != nil {
						infoIface := new(InterfacesInformation)
						infoIface.Iface = iface.Name
						infoIface.IpNet = ipNet
						infoIface.Ip = &iip
						infoIfaces = append(infoIfaces, infoIface)
					} else if !iip.IsLoopback() && !iip.IsPrivate() && iip.To4() != nil {
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

// Return first free port available PORT_START < PORT_RANGE
func getFreePort() (int, error) {
	for i := portStart; i < PORT_RANGE; i++ {
		a, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", i))
		if err == nil {
			var l *net.TCPListener
			if l, err = net.ListenTCP("tcp", a); err == nil {
				defer l.Close()
				return l.Addr().(*net.TCPAddr).Port, nil
			}
		}
	}
	return 0, fmt.Errorf("Couldn't find open port")
}

// use to find interface bridge that use public ip.
// @ip string: this is a private ip this is used for identifying private bridge.
func findingPublicBridge(ip string) string {
	infoIfaces := creatreInterfacesInformation(ip)
	for _, infoIface := range infoIfaces {
		if !infoIface.Ip.IsLoopback() && !infoIface.Ip.IsPrivate() {
			return infoIface.Iface
		}
	}
	return ""
}

// Use to find interface bridge that use the current private ip.
func findingPrivateBridge(ip string) string {
	infoIfaces := creatreInterfacesInformation(ip)
	for _, infoIface := range infoIfaces {
		if infoIface.IpNet.Contains(net.ParseIP(ip)) {
			return infoIface.Iface
		}
	}
	return ""
}

// Need to find bridge on proper interfaces.
func createDNATConfig(ip string) (*DNATConfig, error) {
	ifacePrivate := findingPrivateBridge(ip)
	if ifacePrivate == "" {
		return nil, fmt.Errorf("error no interface for: %s", ip)
	}
	ifacePublic := findingPublicBridge(ip)
	if ifacePublic == "" {
		return nil, fmt.Errorf("error no public bridge interface")

	}
	cfg := new(DNATConfig)
	cfg.Interface = ifacePublic
	cfg.SubNetworkInterface = ifacePrivate
	cfg.DestIP = ip
	return cfg, nil
}

// Remove existing prerouting from iptables base on ip and port.
func removePrerouting(ip, port string) error {
	out, err := exec.Command("/usr/bin/env", "sh", "-c", fmt.Sprintf("iptables -t nat -S | grep %s:%s", ip, port)).Output()
	if string(out) != "" && err != nil {
		return fmt.Errorf("error couldn't execute command: %s", err)
	}
	if string(out) == "" {
		logger.Printf("no rules in iptables for %s:%s", ip, port)
		return nil
	}

	trim := strings.TrimSpace(string(out))
	logger.Debugf("result: [%s]", trim)
	cmd_string := fmt.Sprintf("iptables -t nat %s", "-D"+string(out[2:]))
	logger.Debugf("removed prerouting rules: %s", cmd_string)
	if err := exec.Command("/usr/bin/env", "sh", "-c", cmd_string).Run(); err != nil {
		return fmt.Errorf("error couldn't execute command: %s", err)
	}
	return nil
}

// Create routing with iptables to port forward.
func iptablesPrerouting(cfg *DNATConfig, gameport string) error {
	port, err := getFreePort()
	if err != nil {
		return err
	}
	if usedPort[port] {
		logger.Printf("for %s already port forwading %s:%s -> %d", cfg.DestIP, cfg.DestIP, gameport, port)
		return nil
	}
	if err = removePrerouting(cfg.DestIP, gameport); err != nil {
		return fmt.Errorf("remove prerouting: %s", err)
	}
	command_string := fmt.Sprintf("iptables -t nat -A PREROUTING -i %s -p tcp -m tcp --dport %d -j DNAT --to-destination %s:%s", cfg.Interface, port, cfg.DestIP, gameport)
	if err := exec.Command("/usr/bin/env", "sh", "-c", command_string).Run(); err != nil {
		return err
	}
	logger.Debugf("port %d added to port used", port)
	usedPort[port] = true
	return nil
}

// Forward port of gameserver to ip range of PORT_START to PORT_RANGE
func ForwardPort(ip, ports string) {
	cfg, err := createDNATConfig(ip)
	if err != nil {
		logger.Errf("%s", err)
		return
	}
	iters := strings.SplitSeq(ports, ",")
	for port := range iters {
		if err := iptablesPrerouting(cfg, port); err != nil {
			logger.Errf("%s", err)
			return
		}
	}
}

// TestingPort is function that will return valid ips and ports if no error.
// @validIPs map[int]bool: int is index of 'ips []string' that is valid if true false empty if not.
// @mapPorts map[int][]string: int is index of 'ips []string' []string is all available gameports.
// @err error: error is non nil if error occured.
func TestingPort(ips []string) (validIPs map[int]bool, mapPorts map[int][]string, err error) {
	mapRet := make(map[int]bool)
	mapPort := make(map[int][]string)
	gameConfig, err := parseYaml()
	if err != nil {
		return nil, nil, err
	}
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("error ips is empty")
	}
	gp := &gameConfig.Gameport
	r := reflect.ValueOf(gp).Elem()
	rt := r.Type()
	for i, ip := range ips {
		count := 0
		timeout := time.Second
		for val := 0; val < rt.NumField(); val++ {
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
				ports = make([]string, rt.NumField())
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
