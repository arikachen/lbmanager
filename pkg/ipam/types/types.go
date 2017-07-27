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

package types

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// like net.IPNet but adds JSON marshalling and unmarshalling
type IPNet net.IPNet

// ParseCIDR takes a string like "10.2.3.1/24" and
// return IPNet with "10.2.3.1" and /24 mask
func ParseCIDR(s string) (*net.IPNet, error) {
	ip, ipn, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}

	ipn.IP = ip
	return ipn, nil
}

func (n IPNet) MarshalJSON() ([]byte, error) {
	return json.Marshal((*net.IPNet)(&n).String())
}

func (n *IPNet) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	tmp, err := ParseCIDR(s)
	if err != nil {
		return err
	}

	*n = IPNet(*tmp)
	return nil
}

// IPConfig contains values necessary to configure an IP address on an interface
type IPConfig struct {
	// IP version, either "4" or "6"
	Version string
	Address net.IPNet
	Gateway net.IP
}

type Route struct {
	Dst net.IPNet
	GW  net.IP
}

func (r *Route) String() string {
	return fmt.Sprintf("%+v", *r)
}

// net.IPNet is not JSON (un)marshallable so this duality is needed
// for our custom IPNet type

// JSON (un)marshallable types
type route struct {
	Dst IPNet  `json:"dst"`
	GW  net.IP `json:"gw,omitempty"`
}

func (r *Route) UnmarshalJSON(data []byte) error {
	rt := route{}
	if err := json.Unmarshal(data, &rt); err != nil {
		return err
	}

	r.Dst = net.IPNet(rt.Dst)
	r.GW = rt.GW
	return nil
}

func (r *Route) MarshalJSON() ([]byte, error) {
	rt := route{
		Dst: IPNet(r.Dst),
		GW:  r.GW,
	}

	return json.Marshal(rt)
}

func prettyPrint(obj interface{}) error {
	data, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
