package historyimports

import (
	"errors"
	"testing"
)

func Test_findHistoryClientTrait(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		want    string
		wantErr error
	}{
		{
			name: "single-line import with one client",
			lines: []string{
				"import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
				"export function test() {}",
			},
			want:    "air_quality_sensor",
			wantErr: nil,
		},
		{
			name: "multi-line import with one client",
			lines: []string{
				"import {",
				"  AirQualitySensorHistoryPromiseClient,",
				"  ListAirQualityHistoryRequest",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
				"export function test() {}",
			},
			want:    "air_quality_sensor",
			wantErr: nil,
		},
		{
			name: "mixed single-line and multi-line imports with same client",
			lines: []string{
				"import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
				"import {",
				"  ListAirQualityHistoryRequest,",
				"  GetAirQualityHistoryResponse",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';",
				"export function test() {}",
			},
			want:    "air_quality_sensor",
			wantErr: nil,
		},
		{
			name: "multiple different clients - single-line",
			lines: []string{
				"import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
				"import {MeterHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
			},
			want:    "",
			wantErr: ErrMultipleHistoryClients,
		},
		{
			name: "multiple different clients - multi-line",
			lines: []string{
				"import {",
				"  AirQualitySensorHistoryPromiseClient",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
				"import {",
				"  MeterHistoryPromiseClient",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
			},
			want:    "",
			wantErr: ErrMultipleHistoryClients,
		},
		{
			name: "multiple different clients - mixed single and multi-line",
			lines: []string{
				"import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
				"import {",
				"  MeterHistoryPromiseClient,",
				"  ListMeterReadingHistoryRequest",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
			},
			want:    "",
			wantErr: ErrMultipleHistoryClients,
		},
		{
			name: "no client - single-line import with only messages",
			lines: []string{
				"import {ListAirQualityHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';",
			},
			want:    "",
			wantErr: ErrNoHistoryClient,
		},
		{
			name: "no client - multi-line import with only messages",
			lines: []string{
				"import {",
				"  ListAirQualityHistoryRequest,",
				"  GetAirQualityHistoryResponse",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';",
			},
			want:    "",
			wantErr: ErrNoHistoryClient,
		},
		{
			name: "no history imports",
			lines: []string{
				"import {AirQualitySensorApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/air_quality_sensor_grpc_web_pb';",
				"export function test() {}",
			},
			want:    "",
			wantErr: ErrNoHistoryImports,
		},
		{
			name:    "empty file",
			lines:   []string{},
			want:    "",
			wantErr: ErrNoHistoryImports,
		},
		{
			name: "trait-specific history imports only (not generic)",
			lines: []string{
				"import {TransportHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/transport_history_grpc_web_pb';",
				"export function test() {}",
			},
			want:    "",
			wantErr: ErrNoHistoryImports,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findHistoryClientTrait(tt.lines)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("findHistoryClientTrait() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("findHistoryClientTrait() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("findHistoryClientTrait() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("findHistoryClientTrait() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_collectClientTraitsFromSymbols(t *testing.T) {
	tests := []struct {
		name    string
		symbols []string
		want    []string
	}{
		{
			name:    "single PromiseClient",
			symbols: []string{"AirQualitySensorHistoryPromiseClient"},
			want:    []string{"air_quality_sensor"},
		},
		{
			name:    "single Client",
			symbols: []string{"MeterHistoryClient"},
			want:    []string{"meter"},
		},
		{
			name:    "multiple clients",
			symbols: []string{"AirQualitySensorHistoryPromiseClient", "MeterHistoryClient"},
			want:    []string{"air_quality_sensor", "meter"},
		},
		{
			name:    "mixed with non-client symbols",
			symbols: []string{"ListAirQualityHistoryRequest", "AirQualitySensorHistoryPromiseClient", "GetAirQualityHistoryResponse"},
			want:    []string{"air_quality_sensor"},
		},
		{
			name:    "no client symbols",
			symbols: []string{"ListAirQualityHistoryRequest", "GetAirQualityHistoryResponse"},
			want:    nil,
		},
		{
			name:    "empty",
			symbols: []string{},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectClientTraitsFromSymbols(tt.symbols)
			if len(got) != len(tt.want) {
				t.Errorf("collectClientTraitsFromSymbols() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("collectClientTraitsFromSymbols()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func Test_processSingleLineHistoryImport(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{
			name: "generic history import with client",
			line: "import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
			want: []string{"air_quality_sensor"},
		},
		{
			name: "generic history import with multiple symbols",
			line: "import {ListAirQualityHistoryRequest, AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';",
			want: []string{"air_quality_sensor"},
		},
		{
			name: "generic history import with no client",
			line: "import {ListAirQualityHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';",
			want: nil,
		},
		{
			name: "trait-specific import (not generic history)",
			line: "import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/air_quality_sensor_grpc_web_pb';",
			want: nil,
		},
		{
			name: "not an import line",
			line: "export function test() {",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processSingleLineHistoryImport(tt.line)
			if len(got) != len(tt.want) {
				t.Errorf("processSingleLineHistoryImport() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("processSingleLineHistoryImport()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func Test_hasGenericHistoryImport(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "single-line generic history import",
			line: "import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
			want: true,
		},
		{
			name: "single-line generic history import from _pb",
			line: "import {ListAirQualityHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';",
			want: true,
		},
		{
			name: "trait-specific history import",
			line: "import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/air_quality_sensor_history_grpc_web_pb';",
			want: false,
		},
		{
			name: "trait-specific import (not history)",
			line: "import {AirQualitySensorApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/air_quality_sensor_grpc_web_pb';",
			want: false,
		},
		{
			name: "not an import line",
			line: "export function test() {",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasGenericHistoryImport(tt.line); got != tt.want {
				t.Errorf("hasGenericHistoryImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processMultiLineHistoryImport(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		startIdx int
		want     []string
	}{
		{
			name: "multi-line generic history import with client",
			lines: []string{
				"import {",
				"  AirQualitySensorHistoryPromiseClient,",
				"  ListAirQualityHistoryRequest",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
			},
			startIdx: 0,
			want:     []string{"air_quality_sensor"},
		},
		{
			name: "multi-line generic history import without client",
			lines: []string{
				"import {",
				"  ListAirQualityHistoryRequest,",
				"  GetAirQualityHistoryResponse",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';",
			},
			startIdx: 0,
			want:     nil,
		},
		{
			name: "multi-line trait-specific import (not generic history)",
			lines: []string{
				"import {",
				"  AirQualitySensorHistoryPromiseClient,",
				"  ListAirQualitySensorHistoryRequest",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/air_quality_sensor_history_grpc_web_pb';",
			},
			startIdx: 0,
			want:     nil,
		},
		{
			name: "not a multi-line import start",
			lines: []string{
				"export function test() {",
				"  return true;",
				"}",
			},
			startIdx: 0,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processMultiLineHistoryImport(tt.lines, tt.startIdx)
			if len(got) != len(tt.want) {
				t.Errorf("processMultiLineHistoryImport() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("processMultiLineHistoryImport()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func Test_fileHasGenericHistoryImports(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  bool
	}{
		{
			name: "has single-line generic history import",
			lines: []string{
				"import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
				"export function test() {}",
			},
			want: true,
		},
		{
			name: "has multi-line generic history import",
			lines: []string{
				"import {",
				"  AirQualitySensorHistoryPromiseClient",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
			},
			want: true,
		},
		{
			name: "only trait-specific imports",
			lines: []string{
				"import {AirQualitySensorHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/air_quality_sensor_history_grpc_web_pb';",
				"export function test() {}",
			},
			want: false,
		},
		{
			name: "no imports at all",
			lines: []string{
				"export function test() {",
				"  return true;",
				"}",
			},
			want: false,
		},
		{
			name:  "empty file",
			lines: []string{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileHasGenericHistoryImports(tt.lines); got != tt.want {
				t.Errorf("fileHasGenericHistoryImports() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasGenericHistoryImportMultiLine(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		startIdx int
		want     bool
	}{
		{
			name: "multi-line generic history import",
			lines: []string{
				"import {",
				"  AirQualitySensorHistoryPromiseClient",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';",
			},
			startIdx: 0,
			want:     true,
		},
		{
			name: "multi-line generic history import from _pb",
			lines: []string{
				"import {",
				"  ListAirQualityHistoryRequest",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';",
			},
			startIdx: 0,
			want:     true,
		},
		{
			name: "multi-line trait-specific history import",
			lines: []string{
				"import {",
				"  AirQualitySensorHistoryPromiseClient",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/air_quality_sensor_history_grpc_web_pb';",
			},
			startIdx: 0,
			want:     false,
		},
		{
			name: "multi-line non-history import",
			lines: []string{
				"import {",
				"  AirQualitySensorApiPromiseClient",
				"} from '@smart-core-os/sc-bos-ui-gen/proto/air_quality_sensor_grpc_web_pb';",
			},
			startIdx: 0,
			want:     false,
		},
		{
			name: "not a multi-line import",
			lines: []string{
				"export function test() {",
				"  return true;",
				"}",
			},
			startIdx: 0,
			want:     false,
		},
		{
			name: "incomplete multi-line import (no end)",
			lines: []string{
				"import {",
				"  AirQualitySensorHistoryPromiseClient",
			},
			startIdx: 0,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasGenericHistoryImportMultiLine(tt.lines, tt.startIdx); got != tt.want {
				t.Errorf("hasGenericHistoryImportMultiLine() = %v, want %v", got, tt.want)
			}
		})
	}
}
