package uiproto

import (
	"testing"
)

func TestFixJSImports(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single level relative traits import",
			input: `var traits_foo_pb = require('./traits/foo_pb.js');`,
			want:  `var traits_foo_pb = require('@smart-core-os/sc-api-grpc-web/traits/foo_pb.js');`,
		},
		{
			name:  "multi level relative traits import",
			input: `var traits_foo_pb = require('../../../traits/foo_pb.js');`,
			want:  `var traits_foo_pb = require('@smart-core-os/sc-api-grpc-web/traits/foo_pb.js');`,
		},
		{
			name:  "types import",
			input: `var types_change_pb = require('./types/change_pb.js');`,
			want:  `var types_change_pb = require('@smart-core-os/sc-api-grpc-web/types/change_pb.js');`,
		},
		{
			name:  "info import",
			input: `var info_pb = require('./info/some_info_pb.js');`,
			want:  `var info_pb = require('@smart-core-os/sc-api-grpc-web/info/some_info_pb.js');`,
		},
		{
			name: "multiple imports in one file",
			input: `var traits_foo_pb = require('./traits/foo_pb.js');
var types_change_pb = require('../types/change_pb.js');
var info_pb = require('./info/some_info_pb.js');`,
			want: `var traits_foo_pb = require('@smart-core-os/sc-api-grpc-web/traits/foo_pb.js');
var types_change_pb = require('@smart-core-os/sc-api-grpc-web/types/change_pb.js');
var info_pb = require('@smart-core-os/sc-api-grpc-web/info/some_info_pb.js');`,
		},
		{
			name:  "no matching imports",
			input: `var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');`,
			want:  `var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');`,
		},
		{
			name: "mixed imports",
			input: `var google_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');
var traits_foo_pb = require('./traits/foo_pb.js');
var local_pb = require('./local_pb.js');`,
			want: `var google_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');
var traits_foo_pb = require('@smart-core-os/sc-api-grpc-web/traits/foo_pb.js');
var local_pb = require('./local_pb.js');`,
		},
		{
			name:  "already fixed import",
			input: `var traits_foo_pb = require('@smart-core-os/sc-api-grpc-web/traits/foo_pb.js');`,
			want:  `var traits_foo_pb = require('@smart-core-os/sc-api-grpc-web/traits/foo_pb.js');`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(fixJSImports([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("fixJSImports() =\n%s\n\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestFixDTSImports(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single level relative traits import",
			input: `import * as traits_foo_pb from './traits/foo_pb';`,
			want:  `import * as traits_foo_pb from '@smart-core-os/sc-api-grpc-web/traits/foo_pb';`,
		},
		{
			name:  "multi level relative traits import",
			input: `import * as traits_foo_pb from '../../../traits/foo_pb';`,
			want:  `import * as traits_foo_pb from '@smart-core-os/sc-api-grpc-web/traits/foo_pb';`,
		},
		{
			name:  "types import",
			input: `import * as types_change_pb from './types/change_pb';`,
			want:  `import * as types_change_pb from '@smart-core-os/sc-api-grpc-web/types/change_pb';`,
		},
		{
			name:  "info import",
			input: `import * as info_pb from './info/some_info_pb';`,
			want:  `import * as info_pb from '@smart-core-os/sc-api-grpc-web/info/some_info_pb';`,
		},
		{
			name: "multiple imports in one file",
			input: `import * as traits_foo_pb from './traits/foo_pb';
import * as types_change_pb from '../types/change_pb';
import * as info_pb from './info/some_info_pb';`,
			want: `import * as traits_foo_pb from '@smart-core-os/sc-api-grpc-web/traits/foo_pb';
import * as types_change_pb from '@smart-core-os/sc-api-grpc-web/types/change_pb';
import * as info_pb from '@smart-core-os/sc-api-grpc-web/info/some_info_pb';`,
		},
		{
			name:  "no matching imports",
			input: `import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb';`,
			want:  `import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb';`,
		},
		{
			name: "mixed imports",
			input: `import * as google_pb from 'google-protobuf/google/protobuf/timestamp_pb';
import * as traits_foo_pb from './traits/foo_pb';
import * as local_pb from './local_pb';`,
			want: `import * as google_pb from 'google-protobuf/google/protobuf/timestamp_pb';
import * as traits_foo_pb from '@smart-core-os/sc-api-grpc-web/traits/foo_pb';
import * as local_pb from './local_pb';`,
		},
		{
			name:  "already fixed import",
			input: `import * as traits_foo_pb from '@smart-core-os/sc-api-grpc-web/traits/foo_pb';`,
			want:  `import * as traits_foo_pb from '@smart-core-os/sc-api-grpc-web/traits/foo_pb';`,
		},
		{
			name:  "comment style import annotation",
			input: `import * as types_change_pb from './types/change_pb'; // proto import: "types/change.proto"`,
			want:  `import * as types_change_pb from '@smart-core-os/sc-api-grpc-web/types/change_pb'; // proto import: "types/change.proto"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(fixDTSImports([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("fixDTSImports() =\n%s\n\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func BenchmarkFixJSImports(b *testing.B) {
	input := []byte(`var jspb = require('google-protobuf');
var goog = jspb;
var traits_foo_pb = require('./traits/foo_pb.js');
var traits_bar_pb = require('../../../traits/bar_pb.js');
var types_change_pb = require('./types/change_pb.js');
var info_pb = require('./info/some_info_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fixJSImports(input)
	}
}

func BenchmarkFixDTSImports(b *testing.B) {
	input := []byte(`import * as jspb from 'google-protobuf'
import * as traits_foo_pb from './traits/foo_pb';
import * as traits_bar_pb from '../../../traits/bar_pb';
import * as types_change_pb from './types/change_pb';
import * as info_pb from './info/some_info_pb';
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb';`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fixDTSImports(input)
	}
}
