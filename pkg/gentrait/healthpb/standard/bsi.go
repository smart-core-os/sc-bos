package standard

import "github.com/smart-core-os/sc-bos/pkg/proto/healthpb"

//goland:noinspection GoSnakeCaseUsage
var (
	// BS5266_1_2016 is the British Standard for emergency lighting of premises.
	BS5266_1_2016 = Register(&healthpb.HealthCheck_ComplianceImpact_Standard{
		DisplayName:  "BS 5266",
		Title:        "BS 5266-1:2016",
		Description:  "Code of practice for the emergency lighting of premises",
		Organization: "BSI",
	})
)
