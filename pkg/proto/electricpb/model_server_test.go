package electricpb

import (
	"context"
	"fmt"
	"log"
)

func ExampleModelServer() {
	mem := NewModel()
	device := NewModelServer(mem)

	client := WrapApi(device)
	settings := WrapMemorySettingsApi(device)

	ctx := context.Background()
	_, err := settings.CreateMode(ctx, &CreateModeRequest{
		Name: "foo",
		Mode: &ElectricMode{
			Title:       "Normal mode",
			Description: "Normal mode",
			Segments: []*ElectricMode_Segment{
				{Magnitude: 1},
			},
			Normal: true,
		},
	})
	if err != nil {
		log.Println("create mode failed:", err)
		return
	}

	_, err = client.ClearActiveMode(ctx, &ClearActiveModeRequest{
		Name: "foo",
	})
	if err != nil {
		log.Println("clear mode failed:", err)
	}

	mode, err := client.GetActiveMode(ctx, &GetActiveModeRequest{Name: "foo"})
	if err != nil {
		log.Println("GetActiveMode failed:", err)
		return
	}
	fmt.Println(mode.Title)
	// Output: Normal mode
}
