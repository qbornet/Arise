package run

import (
	lib "autofwd/src/internal"
	"autofwd/src/logger"
	"fmt"
	"regexp"
	"strings"
)

const FILE_DNSMASQ = "/var/lib/misc/dnsmasq.leases"

func checkGamePortOpened(ips []string) ([]string, error) {
	if len(ips) == 0 {
		return nil, fmt.Errorf("no ips available")
	}
	mapIps, mapsPort, err := TestingPort(ips)
	if err != nil {
		return nil, err
	}
	i := 0
	valid := make([]string, len(mapIps))
	for idx, v := range mapIps {
		if v {
			var ports string
			ports = strings.Join(mapsPort[idx], ",")
			valid[i] = fmt.Sprintf("%s:%s", ips[idx], ports)
			i++
		}
	}
	return valid, nil
}

func Start() error {
	var ips []string

	r, err := regexp.Compile(`[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`)
	if err != nil {
		logger.Fatalf("couldn't compile regexp: %s", err)
	}
	rw, err := lib.OpenFile(FILE_DNSMASQ)
	if err != nil {
		logger.Fatalf("%s", err)
	}
	for true {
		if rw.Available() == 0 {
			rw, err = lib.OpenFile(FILE_DNSMASQ)
			if err != nil {
				logger.Fatalf("%s", err)
			}
		}
		line, _, err := rw.ReadLine()
		if err != nil {
			// Continue to check ip port 25565 is port forward or not.
			validIps, err := checkGamePortOpened(ips)
			if err != nil {
				logger.Errf("game port failure: %s", err)
				continue
			}
			// Need to ip forward from their for valid ip.
			for _, ip := range validIps {
				infos := strings.Split(ip, ":")
				ForwardPort(infos[0], infos[1])
			}
		}
		ip := r.FindString(string(line))
		if ip != "" {
			ips = append(ips, ip)
			continue
		}
		/*
			validIps, err := checkGamePortOpened(ips)
			if err != nil {
				logger.Errf("game port failure: %s", err)
				continue
			}
			// Need to ip forward from their
			for _, ip := range validIps {
				infos := strings.Split(ip, ":")
				ForwardPort(infos[0], infos[1])
			}
		*/
	}
	return nil
}
