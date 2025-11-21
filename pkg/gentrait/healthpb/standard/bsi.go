package standard

import (
	"github.com/vanti-dev/sc-bos/pkg/gen"
)

//goland:noinspection GoSnakeCaseUsage
var (
	// BS5266_1_2016 is the British Standard for emergency lighting of premises.
	BS5266_1_2016 = Register(&gen.HealthCheck_ComplianceImpact_Standard{
		DisplayName:  "BS 5266",
		Title:        "BS 5266-1:2016",
		Description:  "Code of practice for the emergency lighting of premises",
		Organization: "BSI",
	})
)
