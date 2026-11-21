package tools

import (
	"context"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type postureCheckAPI interface {
	ListPostureChecks(ctx context.Context) ([]netbird.PostureCheck, error)
}

type listPostureChecksAPI interface {
	postureCheckAPI
	policyAPI
}

type listPostureChecksInput struct{}

type listPostureChecksOutput struct {
	PostureChecks []postureCheckOutput `json:"posture_checks"`
}

type postureCheckOutput struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Description    string               `json:"description,omitempty"`
	Checks         postureChecksOutput  `json:"checks"`
	UsedByPolicies []postureCheckPolicy `json:"used_by_policies"`
}

type postureCheckPolicy struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type postureChecksOutput struct {
	NBVersion        *nbVersionOutput        `json:"nb_version,omitempty"`
	OSVersion        *osVersionOutput        `json:"os_version,omitempty"`
	GeoLocation      *geoLocationOutput      `json:"geo_location,omitempty"`
	PeerNetworkRange *peerNetworkRangeOutput `json:"peer_network_range,omitempty"`
	Process          *processCheckOutput     `json:"process,omitempty"`
	Certificate      *certificateOutput      `json:"certificate,omitempty"`
}

type nbVersionOutput struct {
	MinVersion string `json:"min_version"`
}

type osVersionOutput struct {
	Android       string `json:"android,omitempty"`
	Darwin        string `json:"darwin,omitempty"`
	IOS           string `json:"ios,omitempty"`
	LinuxKernel   string `json:"linux_kernel,omitempty"`
	WindowsKernel string `json:"windows_kernel,omitempty"`
}

type geoLocationOutput struct {
	Action    string           `json:"action"`
	Locations []locationOutput `json:"locations"`
}

type locationOutput struct {
	CountryCode string `json:"country_code"`
	CityName    string `json:"city_name,omitempty"`
}

type peerNetworkRangeOutput struct {
	Action string   `json:"action"`
	Ranges []string `json:"ranges"`
}

type processCheckOutput struct {
	Processes []processOutput `json:"processes"`
}

type processOutput struct {
	LinuxPath   string `json:"linux_path,omitempty"`
	MacPath     string `json:"mac_path,omitempty"`
	WindowsPath string `json:"windows_path,omitempty"`
}

type certificateOutput struct {
	CACertificatesCount int `json:"ca_certificates_count"`
}

func listPostureChecks(c listPostureChecksAPI) (*mcp.Tool, mcp.ToolHandlerFor[listPostureChecksInput, listPostureChecksOutput]) {
	tool := &mcp.Tool{
		Name:        "list_posture_checks",
		Description: "Lists all NetBird posture checks with their configured checks (NetBird and OS minimum versions, geo location, peer network ranges, processes, certificate count) and the policies that require each check. A posture check only takes effect through the policies listed.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, _ listPostureChecksInput) (*mcp.CallToolResult, listPostureChecksOutput, error) {
		var (
			checks   []netbird.PostureCheck
			policies []netbird.Policy
		)
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() (err error) {
			checks, err = c.ListPostureChecks(gctx)
			return err
		})
		g.Go(func() (err error) {
			policies, err = c.ListPolicies(gctx)
			return err
		})
		if err := g.Wait(); err != nil {
			return nil, listPostureChecksOutput{}, err
		}
		return nil, toListPostureChecksOutput(checks, policies), nil
	}
	return tool, handler
}

func toListPostureChecksOutput(checks []netbird.PostureCheck, policies []netbird.Policy) listPostureChecksOutput {
	out := listPostureChecksOutput{PostureChecks: make([]postureCheckOutput, 0, len(checks))}
	for _, pc := range checks {
		used := make([]postureCheckPolicy, 0)
		for _, p := range policies {
			if slices.Contains(p.SourcePostureChecks, pc.ID) {
				used = append(used, postureCheckPolicy{ID: p.ID, Name: p.Name, Enabled: p.Enabled})
			}
		}
		out.PostureChecks = append(out.PostureChecks, postureCheckOutput{
			ID: pc.ID, Name: pc.Name, Description: pc.Description,
			Checks: toPostureChecksOutput(pc.Checks), UsedByPolicies: used,
		})
	}
	return out
}

func toPostureChecksOutput(c netbird.PostureChecks) postureChecksOutput {
	var out postureChecksOutput
	if c.NBVersionCheck != nil {
		out.NBVersion = &nbVersionOutput{MinVersion: c.NBVersionCheck.MinVersion}
	}
	if o := c.OSVersionCheck; o != nil {
		osOut := &osVersionOutput{}
		if o.Android != nil {
			osOut.Android = o.Android.MinVersion
		}
		if o.Darwin != nil {
			osOut.Darwin = o.Darwin.MinVersion
		}
		if o.IOS != nil {
			osOut.IOS = o.IOS.MinVersion
		}
		if o.Linux != nil {
			osOut.LinuxKernel = o.Linux.MinKernelVersion
		}
		if o.Windows != nil {
			osOut.WindowsKernel = o.Windows.MinKernelVersion
		}
		out.OSVersion = osOut
	}
	if g := c.GeoLocationCheck; g != nil {
		locs := make([]locationOutput, 0, len(g.Locations))
		for _, l := range g.Locations {
			locs = append(locs, locationOutput{CountryCode: l.CountryCode, CityName: l.CityName})
		}
		out.GeoLocation = &geoLocationOutput{Action: g.Action, Locations: locs}
	}
	if r := c.PeerNetworkRangeCheck; r != nil {
		out.PeerNetworkRange = &peerNetworkRangeOutput{Action: r.Action, Ranges: nonNil(r.Ranges)}
	}
	if p := c.ProcessCheck; p != nil {
		procs := make([]processOutput, 0, len(p.Processes))
		for _, pr := range p.Processes {
			procs = append(procs, processOutput{LinuxPath: pr.LinuxPath, MacPath: pr.MacPath, WindowsPath: pr.WindowsPath})
		}
		out.Process = &processCheckOutput{Processes: procs}
	}
	if c.CertificateCheck != nil {
		out.Certificate = &certificateOutput{CACertificatesCount: len(c.CertificateCheck.CACertificates)}
	}
	return out
}
