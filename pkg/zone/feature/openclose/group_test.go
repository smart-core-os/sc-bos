package openclose

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

func Test_mergeOpenClosePositions(t *testing.T) {
	err := errors.New("expected error")

	tests := []struct {
		name    string
		in      []value
		want    *traits.OpenClosePositions
		wantErr bool
	}{
		// simple cases
		{"nil", nil, nil, true},
		{"[]", []value{}, nil, true},
		{"[nil]", []value{{}}, ocp(), false},
		{"[err]", []value{{err: err}}, nil, true},
		{"[{}, err]", []value{val(), {err: err}}, nil, true},
		{"[{10%}]", []value{val(pos(10))}, ocp(pos(10)), false},
		// directions
		{"[{10%},{30%}]", []value{val(pos(10)), val(pos(30))}, ocp(pos(20)), false},
		{"[{10% up},{30% down}]", []value{val(up(10)), val(down(30))}, ocp(up(10), down(30)), false},
		{"[{30% down},{10% up}]", []value{val(down(30)), val(up(10))}, ocp(up(10), down(30)), false},
		{"[{10% up},{30% up}]", []value{val(up(10)), val(up(30))}, ocp(up(20)), false},
		{"[{10% up,90% down},{30% up,70% down}]", []value{val(up(10), down(90)), val(up(30), down(70))}, ocp(up(20), down(80)), false},
		// resistances
		{"[{10% slow}]", []value{val(pos(10).Slow())}, ocp(pos(10).Slow()), false},
		{"[{10% rm}]", []value{val(pos(10).RM())}, ocp(pos(10).RM()), false},
		{"[{10% held}]", []value{val(pos(10).Held())}, ocp(pos(10).Held()), false},
		{"[{10%     },{10%     }]", []value{val(pos(10)), val(pos(10))}, ocp(pos(10)), false},
		{"[{10%     },{10% slow}]", []value{val(pos(10)), val(pos(10).Slow())}, ocp(pos(10).Slow()), false},
		{"[{10%     },{10% rm  }]", []value{val(pos(10)), val(pos(10).RM())}, ocp(pos(10).RM()), false},
		{"[{10%     },{10% held}]", []value{val(pos(10)), val(pos(10).Held())}, ocp(pos(10).Held()), false},
		{"[{10% slow},{10%     }]", []value{val(pos(10).Slow()), val(pos(10))}, ocp(pos(10).Slow()), false},
		{"[{10% slow},{10% slow}]", []value{val(pos(10).Slow()), val(pos(10).Slow())}, ocp(pos(10).Slow()), false},
		{"[{10% slow},{10% rm  }]", []value{val(pos(10).Slow()), val(pos(10).RM())}, ocp(pos(10).RM()), false},
		{"[{10% slow},{10% held}]", []value{val(pos(10).Slow()), val(pos(10).Held())}, ocp(pos(10).Held()), false},
		{"[{10% rm  },{10%     }]", []value{val(pos(10).RM()), val(pos(10))}, ocp(pos(10).RM()), false},
		{"[{10% rm  },{10% slow}]", []value{val(pos(10).RM()), val(pos(10).Slow())}, ocp(pos(10).RM()), false},
		{"[{10% rm  },{10% rm  }]", []value{val(pos(10).RM()), val(pos(10).RM())}, ocp(pos(10).RM()), false},
		{"[{10% rm  },{10% held}]", []value{val(pos(10).RM()), val(pos(10).Held())}, ocp(pos(10).Held()), false},
		{"[{10% held},{10%     }]", []value{val(pos(10).Held()), val(pos(10))}, ocp(pos(10).Held()), false},
		{"[{10% held},{10% slow}]", []value{val(pos(10).Held()), val(pos(10).Slow())}, ocp(pos(10).Held()), false},
		{"[{10% held},{10% rm  }]", []value{val(pos(10).Held()), val(pos(10).RM())}, ocp(pos(10).Held()), false},
		{"[{10% held},{10% held}]", []value{val(pos(10).Held()), val(pos(10).Held())}, ocp(pos(10).Held()), false},
		// combine it all together
		{"[{10% up slow,90% down},{30% up,70% down held}]", []value{val(up(10).Slow(), down(90)), val(up(30), down(70).Held())}, ocp(up(20).Slow(), down(80).Held()), false},
		// preset tests
		{"[{open}]", []value{valPres("open")}, ocpPres("open"), false},
		{"[{nil },{open},{nil }]", []value{valPres(""), valPres("open"), valPres("")}, ocpPres("open"), false},
		{"[{open},{open},{open}]", []value{valPres("open"), valPres("open"), valPres("open")}, ocpPres("open"), false},
		{"[{shut},{open},{nil }]", []value{valPres("shut"), valPres("open"), valPres("")}, ocpPres(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeOpenClosePositions(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeOpenClosePositions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("mergeOpenClosePositions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func pos(val float32) *position {
	return &position{OpenClosePosition: &traits.OpenClosePosition{OpenPercent: val}}
}

func up(val float32) *position {
	return pos(val).Up()
}

func down(val float32) *position {
	return pos(val).Down()
}

func ocp(p ...*position) *traits.OpenClosePositions {
	ocp := &traits.OpenClosePositions{}
	for _, v := range p {
		ocp.States = append(ocp.States, v.OpenClosePosition)
	}
	return ocp
}

func val(p ...*position) value {
	return value{val: ocp(p...)}
}

func valPres(name string) value {
	return value{val: ocpPres(name)}
}

func ocpPres(name string) *traits.OpenClosePositions {
	var preset *traits.OpenClosePositions_Preset
	if name != "" {
		preset = &traits.OpenClosePositions_Preset{Name: name}
	}
	return &traits.OpenClosePositions{Preset: preset}
}

type position struct {
	*traits.OpenClosePosition
}

func (p *position) Val(val float32) *position {
	p.OpenPercent = val
	return p
}

func (p *position) Dir(dir traits.OpenClosePosition_Direction) *position {
	p.Direction = dir
	return p
}

func (p *position) Up() *position {
	return p.Dir(traits.OpenClosePosition_UP)
}

func (p *position) Down() *position {
	return p.Dir(traits.OpenClosePosition_DOWN)
}

func (p *position) Res(res traits.OpenClosePosition_Resistance) *position {
	p.Resistance = res
	return p
}

func (p *position) Held() *position {
	return p.Res(traits.OpenClosePosition_HELD)
}

func (p *position) Slow() *position {
	return p.Res(traits.OpenClosePosition_SLOW)
}

func (p *position) RM() *position {
	return p.Res(traits.OpenClosePosition_REDUCED_MOTION)
}
