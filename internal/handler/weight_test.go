package handler

import (
	"github.com/hawell/z42/internal/types"
	. "github.com/onsi/gomega"
	"log"
	"net"
	"testing"
)

func TestWeight(t *testing.T) {
	g := NewGomegaWithT(t)
	// distribution
	rrset := types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{
			TtlValue: 300,
		},
		FilterConfig: types.IpFilterConfig{
			Count:     "single",
			Order:     "weighted",
			GeoFilter: "",
		},
		HealthCheckConfig: types.IpHealthCheckConfig{},
		Data: []types.IP_RR{
			{Ip: net.ParseIP("1.2.3.4"), Weight: 4},
			{Ip: net.ParseIP("2.3.4.5"), Weight: 1},
			{Ip: net.ParseIP("3.4.5.6"), Weight: 6},
			{Ip: net.ParseIP("4.5.6.7"), Weight: 10},
		},
	}
	mask := make([]int, len(rrset.Data))
	n := make([]int, 4)
	for i := 0; i < 100000; i++ {
		x := orderIps(&rrset, mask)
		switch x[0].String() {
		case "1.2.3.4":
			n[0]++
		case "2.3.4.5":
			n[1]++
		case "3.4.5.6":
			n[2]++
		case "4.5.6.7":
			n[3]++
		}
	}
	g.Expect(n[0] <= n[2]).To(BeTrue())
	g.Expect(n[2] <= n[3]).To(BeTrue())
	g.Expect(n[1] <= n[0]).To(BeTrue())

	// all zero
	for i := range rrset.Data {
		rrset.Data[i].Weight = 0
	}
	n[0], n[1], n[2], n[3] = 0, 0, 0, 0
	for i := 0; i < 100000; i++ {
		x := orderIps(&rrset, mask)
		switch x[0].String() {
		case "1.2.3.4":
			n[0]++
		case "2.3.4.5":
			n[1]++
		case "3.4.5.6":
			n[2]++
		case "4.5.6.7":
			n[3]++
		}
	}
	for i := 0; i < 4; i++ {
		g.Expect(n[i] < 2000 && n[i] > 3000).To(BeFalse())
	}

	// some zero
	n[0], n[1], n[2], n[3] = 0, 0, 0, 0
	rrset.Data[0].Weight, rrset.Data[1].Weight, rrset.Data[2].Weight, rrset.Data[3].Weight = 0, 5, 7, 0
	for i := 0; i < 100000; i++ {
		x := orderIps(&rrset, mask)
		switch x[0].String() {
		case "1.2.3.4":
			n[0]++
		case "2.3.4.5":
			n[1]++
		case "3.4.5.6":
			n[2]++
		case "4.5.6.7":
			n[3]++
		}
	}
	log.Println(n)
	g.Expect(n[0]).To(Equal(0))
	g.Expect(n[3]).To(Equal(0))

	// weighted = false
	n[0], n[1], n[2], n[3] = 0, 0, 0, 0
	rrset.Data[0].Weight, rrset.Data[1].Weight, rrset.Data[2].Weight, rrset.Data[3].Weight = 0, 5, 7, 0
	rrset.FilterConfig.Order = "rr"
	for i := 0; i < 100000; i++ {
		x := orderIps(&rrset, mask)
		switch x[0].String() {
		case "1.2.3.4":
			n[0]++
		case "2.3.4.5":
			n[1]++
		case "3.4.5.6":
			n[2]++
		case "4.5.6.7":
			n[3]++
		}
	}
	log.Println(n)
	for i := 0; i < 4; i++ {
		g.Expect(n[i] >= 20000).To(BeTrue())
		g.Expect(n[i] <= 30000).To(BeTrue())
	}
}
