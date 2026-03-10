package electricpb

import (
	"context"
	"fmt"
	"log"

	"github.com/smart-core-os/sc-api/go/traits"
)

func ExampleModelServer() {
	mem := NewModel()
	device := NewModelServer(mem)

	client := WrapApi(device)
	settings := WrapMemorySettingsApi(device)

	ctx := context.Background()
	_, err := settings.CreateMode(ctx, &CreateModeRequest{
		Name: "foo",
		Mode: &traits.ElectricMode{
			Title:       "Normal mode",
			Description: "Normal mode",
			Segments: []*traits.ElectricMode_Segment{
				{Magnitude: 1},
			},
			Normal: true,
		},
	})
	if err != nil {
		log.Println("create mode failed:", err)
		return
	}

	_, err = client.ClearActiveMode(ctx, &traits.ClearActiveModeRequest{
		Name: "foo",
	})
	if err != nil {
		log.Println("clear mode failed:", err)
	}

	mode, err := client.GetActiveMode(ctx, &traits.GetActiveModeRequest{Name: "foo"})
	if err != nil {
		log.Println("GetActiveMode failed:", err)
		return
	}
	fmt.Println(mode.Title)
	// Output: Normal mode
}
