package netbird

import (
	"context"
	"fmt"
)

type PostureCheck struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Checks      PostureChecks `json:"checks"`
}

type PostureChecks struct {
	NBVersionCheck        *MinVersionCheck       `json:"nb_version_check"`
	OSVersionCheck        *OSVersionCheck        `json:"os_version_check"`
	GeoLocationCheck      *GeoLocationCheck      `json:"geo_location_check"`
	PeerNetworkRangeCheck *PeerNetworkRangeCheck `json:"peer_network_range_check"`
	ProcessCheck          *ProcessCheck          `json:"process_check"`
	CertificateCheck      *CertificateCheck      `json:"certificate_check"`
}

type MinVersionCheck struct {
	MinVersion string `json:"min_version"`
}

type MinKernelVersionCheck struct {
	MinKernelVersion string `json:"min_kernel_version"`
}

type OSVersionCheck struct {
	Android *MinVersionCheck       `json:"android"`
	Darwin  *MinVersionCheck       `json:"darwin"`
	IOS     *MinVersionCheck       `json:"ios"`
	Linux   *MinKernelVersionCheck `json:"linux"`
	Windows *MinKernelVersionCheck `json:"windows"`
}

type Location struct {
	CountryCode string `json:"country_code"`
	CityName    string `json:"city_name"`
}

type GeoLocationCheck struct {
	Locations []Location `json:"locations"`
	Action    string     `json:"action"`
}

type PeerNetworkRangeCheck struct {
	Ranges []string `json:"ranges"`
	Action string   `json:"action"`
}

type Process struct {
	LinuxPath   string `json:"linux_path"`
	MacPath     string `json:"mac_path"`
	WindowsPath string `json:"windows_path"`
}

type ProcessCheck struct {
	Processes []Process `json:"processes"`
}

type CertificateCheck struct {
	CACertificates []string `json:"ca_certificates"`
}

func (c *Client) ListPostureChecks(ctx context.Context) ([]PostureCheck, error) {
	var checks []PostureCheck
	if err := c.get(ctx, &checks, nil, "posture-checks"); err != nil {
		return nil, fmt.Errorf("list posture checks: %w", err)
	}
	return checks, nil
}
