package tools

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

func TestGroupNamesByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		groups []netbird.Group
		want   map[string]string
	}{
		{name: "nil", want: map[string]string{}},
		{
			name:   "maps ids to names",
			groups: []netbird.Group{{ID: "g1", Name: "devs"}, {ID: "g2", Name: "ops"}},
			want:   map[string]string{"g1": "devs", "g2": "ops"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := groupNamesByID(tt.groups)
			if got == nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupNamesByID = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestToGroupRefs(t *testing.T) {
	t.Parallel()
	names := map[string]string{"g1": "devs", "g2": "ops"}
	tests := []struct {
		name  string
		ids   []string
		names map[string]string
		want  []refOutput
	}{
		{name: "nil ids", names: names, want: []refOutput{}},
		{name: "nil names", ids: []string{"g1"}, want: []refOutput{{ID: "g1"}}},
		{
			name:  "keeps order",
			ids:   []string{"g2", "g1"},
			names: names,
			want:  []refOutput{{ID: "g2", Name: "ops"}, {ID: "g1", Name: "devs"}},
		},
		{
			name:  "unknown id has empty name",
			ids:   []string{"gone", "g1"},
			names: names,
			want:  []refOutput{{ID: "gone"}, {ID: "g1", Name: "devs"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toGroupRefs(tt.ids, tt.names)
			if got == nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toGroupRefs = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestToGroupOutputs(t *testing.T) {
	t.Parallel()
	if got := toGroupOutputs(nil); got == nil || len(got) != 0 {
		t.Errorf("nil input = %#v, want empty non-nil", got)
	}
	got := toGroupOutputs([]netbird.Group{{ID: "g1", Name: "a", PeersCount: 3}})
	if !reflect.DeepEqual(got, []refOutput{{ID: "g1", Name: "a"}}) {
		t.Errorf("got %+v", got)
	}
}

func TestToLocalFlagsOutput(t *testing.T) {
	t.Parallel()
	if got := toLocalFlagsOutput(nil); got != nil {
		t.Errorf("nil input = %#v, want nil", got)
	}
	on, off := true, false
	in := &netbird.PeerLocalFlags{RosenpassEnabled: &on, DisableDNS: &off}
	got := toLocalFlagsOutput(in)
	if got == nil {
		t.Fatal("got nil, want output")
	}
	if got.RosenpassEnabled == nil || !*got.RosenpassEnabled || got.DisableDNS == nil || *got.DisableDNS {
		t.Errorf("set flags not copied: %+v", got)
	}
	if got.BlockInbound != nil || got.ServerSSHAllowed != nil || got.LazyConnectionEnabled != nil {
		t.Errorf("unset flags not nil: %+v", got)
	}
	in.BlockInbound = &on
	if got.BlockInbound != nil {
		t.Errorf("result aliases input: %+v", got)
	}
}

func TestNonNil(t *testing.T) {
	t.Parallel()
	got := nonNil(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("nil input = %#v, want empty non-nil", got)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "[]" {
		t.Errorf("marshal = %s, want []", b)
	}
	in := []string{"a", "b"}
	if got := nonNil(in); !reflect.DeepEqual(got, in) {
		t.Errorf("non-empty = %#v, want %#v", got, in)
	}
}

func TestPeerNamesByID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		peers []netbird.Peer
		want  map[string]string
	}{
		{name: "nil", want: map[string]string{}},
		{
			name:  "maps ids to names",
			peers: []netbird.Peer{{ID: "p1", Name: "laptop"}, {ID: "p2", Name: "server"}},
			want:  map[string]string{"p1": "laptop", "p2": "server"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := peerNamesByID(tt.peers)
			if got == nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("peerNamesByID = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestToPeerRef(t *testing.T) {
	t.Parallel()
	names := map[string]string{"p1": "laptop"}
	tests := []struct {
		name  string
		id    string
		names map[string]string
		want  string
	}{
		{name: "empty id", id: "", names: names, want: "null"},
		{name: "known", id: "p1", names: names, want: `{"id":"p1","name":"laptop"}`},
		{name: "unknown has no name", id: "gone", names: names, want: `{"id":"gone"}`},
		{name: "nil names", id: "p1", want: `{"id":"p1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ref := toPeerRef(tt.id, tt.names)
			if (tt.want == "null") != (ref == nil) {
				t.Fatalf("toPeerRef = %#v, want %s", ref, tt.want)
			}
			b, err := json.Marshal(ref)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(b) != tt.want {
				t.Errorf("got %s, want %s", b, tt.want)
			}
		})
	}
}
