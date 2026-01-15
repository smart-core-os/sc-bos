// Command test-statusemail tests the [statusemail] package, sending to a real email address.
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/statusemail"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen_statuspb "github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	root := node.New("test")

	var models []*statuspb.Model
	deviceCount := 100
	for i := range deviceCount {
		m := statuspb.NewModel()
		m.UpdateProblem(&gen_statuspb.StatusLog_Problem{Name: "test", Level: gen_statuspb.StatusLog_OFFLINE})
		models = append(models, m)
		client := node.WithClients(gen_statuspb.WrapApi(statuspb.NewModelServer(m)))
		root.Announce(fmt.Sprintf("device-%d", i), node.HasTrait(statuspb.TraitName, client),
			node.HasMetadata(&traits.Metadata{
				Appearance: &traits.Metadata_Appearance{Title: fmt.Sprintf("Device %d", i)},
				Location:   &traits.Metadata_Location{Floor: fmt.Sprintf("Floor %d", i%10), Zone: fmt.Sprintf("Room %d", i%5)},
				Membership: &traits.Metadata_Membership{Subsystem: "bms"},
			}))
	}

	serv := auto.Services{
		Logger: logger,
		Node:   root,
	}
	lifecycle := statusemail.Factory.New(serv)
	defer lifecycle.Stop()
	cfg := `{
  "name": "emails", "type": "statusemail",
  "discoverSources": true,
  "destination": {
    "host": "smtp.gmail.com",
    "from": "Enterprise Wharf <no-reply@enterprisewharf.co.uk>",
    "to": ["Matt Nathan <matt.nathan@vanti.co.uk>"],
    "passwordFile": ".secrets/ew-email-pass",
    "minInterval": "30s"
  }
}`
	_, err = lifecycle.Configure([]byte(cfg))
	if err != nil {
		panic(err)
	}
	_, err = lifecycle.Start()
	if err != nil {
		panic(err)
	}

	// this mimics how drivers work
	time.Sleep(2 * time.Second)
	for _, model := range models {
		model.UpdateProblem(&gen_statuspb.StatusLog_Problem{Name: "test", Level: gen_statuspb.StatusLog_NOMINAL})
	}

	levels := []gen_statuspb.StatusLog_Level{
		gen_statuspb.StatusLog_NOMINAL,
		gen_statuspb.StatusLog_NOTICE,
		gen_statuspb.StatusLog_REDUCED_FUNCTION,
		gen_statuspb.StatusLog_NON_FUNCTIONAL,
		gen_statuspb.StatusLog_OFFLINE,
	}
	for range 100 {
		mi := rand.Int31n(int32(len(models)))
		m := models[mi]
		l := levels[rand.Int31n(int32(len(levels)))]
		n := fmt.Sprintf("device-%d", mi)
		log.Println("updating level for", n, "to", l)
		_, err := m.UpdateProblem(&gen_statuspb.StatusLog_Problem{Name: "test", Level: l, Description: "test message"})
		if err != nil {
			panic(err)
		}
		d := time.Duration(rand.Int31n(10)) * time.Second
		time.Sleep(d)
	}
}
