// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package allocator

import (
	"net"

	"github.com/ArikaChen/lbmanager/pkg/ipam/types"
)

// IPAMConfig represents the IP related network configuration.
type IPAMConfig struct {
	Name       string
	RangeStart net.IP        `json:"rangeStart"`
	RangeEnd   net.IP        `json:"rangeEnd"`
	Subnet     types.IPNet   `json:"subnet"`
	Gateway    net.IP        `json:"gateway"`
	Routes     []types.Route `json:"routes"`
}

func NewIPAMConf(name string, cidr *net.IPNet) *IPAMConfig {
	n := types.IPNet{IP: cidr.IP, Mask: cidr.Mask}
	c := &IPAMConfig{
		Name:   name,
		Subnet: n,
	}
	return c
}

func convertRoutesToCurrent(routes []types.Route) []*types.Route {
	var currentRoutes []*types.Route
	for _, r := range routes {
		currentRoutes = append(currentRoutes, &types.Route{
			Dst: r.Dst,
			GW:  r.GW,
		})
	}
	return currentRoutes
}
