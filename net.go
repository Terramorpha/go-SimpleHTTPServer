package main

import (
	"net"
	"strings"
)

//GetAddress is a simple way to get main ip address
func GetAddress() (addrs []net.IP) {
	o := make([]net.Addr, 0)
	ifaces, err := net.Interfaces()
	if err != nil {
		vPrintf(1, err.Error())
		return nil
	}
	dPrintln(ifaces)
	for i := range ifaces {
		if ifaces[i].Flags&net.FlagLoopback == 0 {
			addrs, err := ifaces[i].Addrs()
			if err != nil {
				vPrintf(1, err.Error())
				continue
			}
			if len(addrs) == 0 {
				continue
			}
			dPrintf("addresses of interface %d: %v\n", i, addrs)
			o = append(o, addrs...)

		}
	}
	if len(o) == 0 {
		//log.Printf("address: %s\n", addr)
	} else {
		for _, v := range o {
			//fmt.Println(v)
			ip := net.ParseIP(strings.Split(v.String(), "/")[0])
			addrs = append(addrs, ip)
			vPrintf(1, "other address: %v\n", ip)

		}
	}
	return

}
