package tools

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

func TestToGroupOutputs(t *testing.T) {
	t.Parallel()
	if got := toGroupOutputs(nil); got == nil || len(got) != 0 {
		t.Errorf("nil input = %#v, want empty non-nil", got)
	}
	got := toGroupOutputs([]netbird.Group{{ID: "g1", Name: "a", PeersCount: 3}})
	if !reflect.DeepEqual(got, []groupOutput{{ID: "g1", Name: "a"}}) {
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
