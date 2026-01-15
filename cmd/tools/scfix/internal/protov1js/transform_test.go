package protov1js

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldProcessFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"js file", "/app/src/test.js", true},
		{"ts file", "/app/src/test.ts", true},
		{"vue file", "/app/src/test.vue", true},
		{"jsx file", "/app/src/test.jsx", true},
		{"tsx file", "/app/src/test.tsx", true},
		{"mjs file", "/app/src/test.mjs", true},
		{"cjs file", "/app/src/test.cjs", true},
		{"d.ts file", "/app/types/test.d.ts", true},
		{"file in node_modules", "/app/node_modules/test.js", false},
		{"file in dist", "/app/dist/test.js", false},
		{"file in .git", "/app/.git/test.js", false},
		{"non-js file", "/app/src/test.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &fakeDirEntry{name: filepath.Base(tt.path), isDir: false}
			got := shouldProcessFile(tt.path, d)
			if got != tt.want {
				t.Errorf("shouldProcessFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestReplaceImportPaths(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		oldImport string
		newImport string
		want      string
	}{
		{
			name:      "ES6 import with single quotes",
			content:   `import {Alert} from '@smart-core-os/sc-bos-ui-gen/proto/alerts_pb';`,
			oldImport: "alerts_pb",
			newImport: "smartcore/bos/alerts/v1/alerts_pb",
			want:      `import {Alert} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alerts/v1/alerts_pb';`,
		},
		{
			name:      "ES6 import with double quotes",
			content:   `import {Alert} from "@smart-core-os/sc-bos-ui-gen/proto/alerts_pb";`,
			oldImport: "alerts_pb",
			newImport: "smartcore/bos/alerts/v1/alerts_pb",
			want:      `import {Alert} from "@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alerts/v1/alerts_pb";`,
		},
		{
			name:      "ES6 import with .js extension",
			content:   `import {Alert} from '@smart-core-os/sc-bos-ui-gen/proto/alerts_pb.js';`,
			oldImport: "alerts_pb",
			newImport: "smartcore/bos/alerts/v1/alerts_pb",
			want:      `import {Alert} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alerts/v1/alerts_pb.js';`,
		},
		{
			name:      "JSDoc typedef",
			content:   `/** @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/meter_pb').MeterReading} MeterReading */`,
			oldImport: "meter_pb",
			newImport: "smartcore/bos/meter/v1/meter_pb",
			want:      `/** @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/meter/v1/meter_pb').MeterReading} MeterReading */`,
		},
		{
			name:      "JSDoc type annotation",
			content:   `/** @type {import('@smart-core-os/sc-bos-ui-gen/proto/alerts_pb').Alert.Query.AsObject} */`,
			oldImport: "alerts_pb",
			newImport: "smartcore/bos/alerts/v1/alerts_pb",
			want:      `/** @type {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alerts/v1/alerts_pb').Alert.Query.AsObject} */`,
		},
		{
			name:      "grpc_web_pb import",
			content:   `import {AccountApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/account_grpc_web_pb';`,
			oldImport: "account_grpc_web_pb",
			newImport: "smartcore/bos/account/v1/account_grpc_web_pb",
			want:      `import {AccountApiPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/account/v1/account_grpc_web_pb';`,
		},
		{
			name: "multiple imports in file",
			content: `import {Alert} from '@smart-core-os/sc-bos-ui-gen/proto/alerts_pb';
/** @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/alerts_pb').Alert} Alert */`,
			oldImport: "alerts_pb",
			newImport: "smartcore/bos/alerts/v1/alerts_pb",
			want: `import {Alert} from '@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alerts/v1/alerts_pb';
/** @typedef {import('@smart-core-os/sc-bos-ui-gen/proto/smartcore/bos/alerts/v1/alerts_pb').Alert} Alert */`,
		},
		{
			name:      "should not affect external imports",
			content:   `import {OnOff} from '@smart-core-os/sc-api-grpc-web/traits/on_off_pb.js';`,
			oldImport: "alerts_pb",
			newImport: "smartcore/bos/alerts/v1/alerts_pb",
			want:      `import {OnOff} from '@smart-core-os/sc-api-grpc-web/traits/on_off_pb.js';`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceImportPaths(tt.content, tt.oldImport, tt.newImport)
			if got != tt.want {
				t.Errorf("replaceImportPaths() mismatch:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f *fakeDirEntry) Name() string               { return f.name }
func (f *fakeDirEntry) IsDir() bool                { return f.isDir }
func (f *fakeDirEntry) Type() os.FileMode          { return 0 }
func (f *fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }
