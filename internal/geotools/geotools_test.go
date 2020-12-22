package geotools

import (
	"fmt"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/pkg/geoip"
	"log"
	"net"
	"strconv"
	"testing"
)

var (
	asnDB     = "../../assets/geoIsp.mmdb"
	countryDB = "../../assets/geoCity.mmdb"
)

func TestGeoIpAutomatic(t *testing.T) {
	sip := [][]string{
		{"212.83.32.45", "DE", "213.95.10.76"},
		{"80.67.163.250", "FR", "62.240.228.4"},
		{"178.18.89.144", "NL", "46.19.36.12"},
		{"206.108.0.43", "CA", "154.11.253.242"},
		{"185.70.144.117", "DE", "213.95.10.76"},
		{"62.220.128.73", "CH", "82.220.3.51"},
	}

	dip := [][]string{
		{"82.220.3.51", "CH"},
		{"192.30.252.225", "US"},
		{"213.95.10.76", "DE"},
		{"94.76.229.204", "GB"},
		{"46.19.36.12", "NL"},
		{"46.30.209.1", "DK"},
		{"91.239.97.26", "SI"},
		{"14.1.44.230", "NZ"},
		{"52.76.214.87", "SG"},
		{"103.31.84.12", "MV"},
		{"212.63.210.241", "SE"},
		{"154.11.253.242", "CA"},
		{"128.139.197.81", "IL"},
		{"194.190.198.13", "RU"},
		{"84.88.14.229", "ES"},
		{"79.110.197.36", "PL"},
		{"175.45.73.66", "AU"},
		{"62.240.228.4", "FR"},
		{"200.238.130.54", "BR"},
		{"13.113.70.195", "JP"},
		{"37.252.235.214", "AT"},
		{"185.87.111.13", "FI"},
		{"52.66.51.117", "IN"},
		{"193.198.233.217", "HR"},
		{"118.67.200.190", "KH"},
		{"103.6.84.107", "HK"},
		{"78.128.211.50", "CZ"},
		{"87.238.39.42", "NO"},
		{"37.148.176.54", "BE"},
	}

	cfg := geoip.Config{
		Enable:    true,
		CountryDB: countryDB,
	}

	g := geoip.NewGeoIp(&cfg)

	for i := range sip {
		dest := new(types.IP_RRSet)
		for j := range dip {
			cc, _ := g.GetCountry(net.ParseIP(dip[j][0]))
			if cc != dip[j][1] {
				t.Fail()
			}
			r := types.IP_RR{
				Ip: net.ParseIP(dip[j][0]),
			}
			dest.Data = append(dest.Data, r)
		}
		dest.Ttl = 100
		mask := make([]int, len(dest.Data))
		mask, err := GetMinimumDistance(g, net.ParseIP(sip[i][0]), dest.Data, mask)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}
		index := 0
		for j, x := range mask {
			if x == types.IpMaskWhite {
				index = j
				break
			}
		}
		log.Println("[DEBUG]", sip[i][0], " ", dest.Data[index].Ip.String())
		if sip[i][2] != dest.Data[index].Ip.String() {
			t.Fail()
		}
	}
}

func TestGetSameCountry(t *testing.T) {
	sip := [][]string{
		{"212.83.32.45", "DE", "1.2.3.4"},
		{"80.67.163.250", "FR", "2.3.4.5"},
		{"154.11.253.242", "", "3.4.5.6"},
		{"127.0.0.1", "", "3.4.5.6"},
	}

	cfg := geoip.Config{
		Enable:    true,
		CountryDB: countryDB,
	}

	g := geoip.NewGeoIp(&cfg)

	for i := range sip {
		var dest types.IP_RRSet
		dest.Data = []types.IP_RR{
			{Ip: net.ParseIP("1.2.3.4"), Country: []string{"DE"}},
			{Ip: net.ParseIP("2.3.4.5"), Country: []string{"FR"}},
			{Ip: net.ParseIP("3.4.5.6"), Country: []string{""}},
		}
		mask := make([]int, len(dest.Data))
		mask, err := GetSameCountry(g, net.ParseIP(sip[i][0]), dest.Data, mask)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}
		index := -1
		for j, x := range mask {
			if x == types.IpMaskWhite {
				index = j
				break
			}
		}
		if index == -1 {
			t.Fail()
		}
		log.Println("[DEBUG]", sip[i][1], sip[i][2], dest.Data[index].Country, dest.Data[index].Ip.String())
		if dest.Data[index].Country[0] != sip[i][1] || dest.Data[index].Ip.String() != sip[i][2] {
			t.Fail()
		}
	}

}

func TestGetSameASN(t *testing.T) {
	sip := []string{
		"212.83.32.45",
		"80.67.163.250",
		"154.11.253.242",
		"127.0.0.1",
	}

	dip := types.IP_RRSet{
		Data: []types.IP_RR{
			{Ip: net.ParseIP("1.2.3.4"), ASN: []uint{47447}},
			{Ip: net.ParseIP("2.3.4.5"), ASN: []uint{20766}},
			{Ip: net.ParseIP("3.4.5.6"), ASN: []uint{852}},
			{Ip: net.ParseIP("4.5.6.7"), ASN: []uint{0}},
		},
	}

	res := [][]string{
		{"47447", "1.2.3.4"},
		{"20766", "2.3.4.5"},
		{"852", "3.4.5.6"},
		{"0", "4.5.6.7"},
	}
	cfg := geoip.Config{
		Enable: true,
		ASNDB:  asnDB,
	}

	g := geoip.NewGeoIp(&cfg)

	for i := range sip {
		mask := make([]int, len(dip.Data))
		mask, err := GetSameASN(g, net.ParseIP(sip[i]), dip.Data, mask)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}
		index := -1
		for j, x := range mask {
			if x == types.IpMaskWhite {
				index = j
				break
			}
		}
		if index == -1 {
			t.Fail()
		}
		if strconv.Itoa(int(dip.Data[index].ASN[0])) != res[i][0] || dip.Data[index].Ip.String() != res[i][1] {
			t.Fail()
		}
	}
}

func TestDisabled(t *testing.T) {
	cfg := geoip.Config{
		Enable:    false,
		CountryDB: countryDB,
		ASNDB:     asnDB,
	}
	g := geoip.NewGeoIp(&cfg)

	_, err := GetMinimumDistance(g, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	if err != geoip.ErrGeoIpDisabled {
		t.Fail()
	}
	_, err = GetSameASN(g, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	if err != geoip.ErrGeoIpDisabled {
		t.Fail()
	}
	_, err = GetSameCountry(g, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	if err != geoip.ErrGeoIpDisabled {
		t.Fail()
	}
}

func TestBadDB(t *testing.T) {
	cfg := geoip.Config{
		Enable:    true,
		CountryDB: "ddd",
		ASNDB:     "ddds",
	}
	g := geoip.NewGeoIp(&cfg)

	_, err := GetMinimumDistance(g, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	if err != geoip.ErrBadDB {
		t.Fail()
	}
	_, err = GetSameASN(g, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	if err != geoip.ErrBadDB {
		t.Fail()
	}
	_, err = GetSameCountry(g, net.ParseIP("1.2.3.4"),
		[]types.IP_RR{{
			Weight:  0,
			Ip:      nil,
			Country: nil,
			ASN:     nil,
		}}, []int{0})
	if err != geoip.ErrBadDB {
		t.Fail()
	}
}
