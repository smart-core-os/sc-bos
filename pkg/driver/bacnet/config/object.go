package config

import (
	"fmt"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

type Object struct {
	ID    ObjectID `json:"id"`
	Name  string   `json:"name,omitempty"`
	Title string   `json:"title,omitempty"`

	COV *COV `json:"COV,omitempty"`

	Trait    trait.Name       `json:"trait,omitempty"`
	Metadata *traits.Metadata `json:"metadata,omitempty"` // applied to any traits created from this object

	Priorities []Priority `json:"priorities,omitempty"`
	Properties []Property `json:"properties,omitempty"`
}

type Priority struct {
	Name  string `json:"name,omitempty"`
	Level uint8  `json:"level,omitempty"`
}

type Property struct {
	Name string     `json:"name,omitempty"`
	ID   PropertyID `json:"id,omitempty"`
}

func (o Object) String() string {
	if o.Name == "" {
		return fmt.Sprintf("%s", o.ID)
	}
	return fmt.Sprintf("%s (%s)", o.Name, o.ID)
}
