//go:build unix

package tarutil

import (
	"archive/tar"
	"bytes"
	"io/fs"
	"syscall"
	"testing"
)

func TestExtractTarGZ_modes(t *testing.T) {
	// clear the umask so the modes the extraction asks for are observable
	oldMask := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(oldMask) })

	tests := []struct {
		name string
		mode int64
		want fs.FileMode
	}{
		{"executable", 0755, 0755},
		{"not executable", 0644, 0644},
		{"world writable", 0777, 0755},
		{"no mode", 0, 0600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := makeTarGZ(t, entry{name: "bin", typeflag: tar.TypeReg, mode: tt.mode, data: "#!/bin/sh\n"})
			root := openRoot(t)

			if err := ExtractTarGZ(bytes.NewReader(data), root, -1); err != nil {
				t.Fatalf("ExtractTarGZ: %v", err)
			}
			info, err := root.Stat("bin")
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if got := info.Mode(); got != tt.want {
				t.Errorf("mode = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("directory", func(t *testing.T) {
		data := makeTarGZ(t, entry{name: "sub", typeflag: tar.TypeDir, mode: 0777})
		root := openRoot(t)

		if err := ExtractTarGZ(bytes.NewReader(data), root, -1); err != nil {
			t.Fatalf("ExtractTarGZ: %v", err)
		}
		info, err := root.Stat("sub")
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if got := info.Mode().Perm(); got != 0755 {
			t.Errorf("mode = %v, want %v", got, fs.FileMode(0755))
		}
	})
}
