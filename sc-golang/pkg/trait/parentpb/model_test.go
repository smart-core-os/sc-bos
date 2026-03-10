package parentpb

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func TestModel_RemoveChildTrait(t *testing.T) {
	t.Run("no traits", func(t *testing.T) {
		model := NewModel()
		expect, _ := model.AddChildTrait("test1", trait.Light, trait.OnOff)
		got := model.RemoveChildTrait("test1")
		if diff := cmp.Diff(expect, got, protocmp.Transform()); diff != "" {
			t.Fatalf("Expected no change child, (-want, +got)\n%v", diff)
		}
	})
	t.Run("remove unsupported traits", func(t *testing.T) {
		model := NewModel()
		expect, _ := model.AddChildTrait("test1", trait.Light, trait.OnOff)
		got := model.RemoveChildTrait("test1", trait.Publication)
		if diff := cmp.Diff(expect, got, protocmp.Transform()); diff != "" {
			t.Fatalf("Expected no change child, (-want, +got)\n%v", diff)
		}
	})
	t.Run("unknown child", func(t *testing.T) {
		model := NewModel()
		model.AddChildTrait("test1", trait.Light, trait.OnOff)
		c := model.RemoveChildTrait("test2", trait.Light)
		if c != nil {
			t.Fatalf("Expected nil child, got %v", c)
		}
	})
	t.Run("remove one", func(t *testing.T) {
		model := NewModel()
		model.AddChildTrait("test1", trait.Light, trait.OnOff)
		c := model.RemoveChildTrait("test1", trait.OnOff)
		if c == nil {
			t.Fatalf("Nil child")
		}
		want := []*traits.Trait{{Name: trait.Light.String()}}
		if !reflect.DeepEqual(c.Traits, want) {
			t.Fatalf("Traits not equal. Want %v, got %v", want, c.Traits)
		}
	})
	t.Run("remove all", func(t *testing.T) {
		model := NewModel()
		model.AddChildTrait("test1", trait.Light, trait.OnOff)
		c := model.RemoveChildTrait("test1", trait.OnOff, trait.Light)
		if c == nil {
			t.Fatalf("Nil child")
		}
		var want []*traits.Trait
		if !reflect.DeepEqual(c.Traits, want) {
			t.Fatalf("Traits not equal. Want %v, got %v", want, c.Traits)
		}
	})
	t.Run("remove all+extra", func(t *testing.T) {
		model := NewModel()
		model.AddChildTrait("test1", trait.Light, trait.OnOff)
		c := model.RemoveChildTrait("test1", trait.OnOff, trait.AirQualitySensor, trait.Light)
		if c == nil {
			t.Fatalf("Nil child")
		}
		var want []*traits.Trait
		if !reflect.DeepEqual(c.Traits, want) {
			t.Fatalf("Traits not equal. Want %v, got %v", want, c.Traits)
		}
	})
}
