package renderer_test

import (
	"testing"

	"github.com/kitsunium/statusline/render/powerline/internal/model"
	"github.com/kitsunium/statusline/render/powerline/internal/renderer"
)

func TestGetOSIcon(t *testing.T) {
	tests := []struct {
		name     string
		osType   model.OSType
		isDocker bool
		want     string
	}{
		{name: "linux", osType: model.OSLinux, isDocker: false, want: renderer.IconLinux},
		{name: "linux in docker", osType: model.OSLinux, isDocker: true, want: renderer.IconDocker},
		{name: "darwin", osType: model.OSDarwin, isDocker: false, want: renderer.IconApple},
		{name: "windows", osType: model.OSWindows, isDocker: false, want: renderer.IconWindows},
		{name: "unknown", osType: model.OSUnknown, isDocker: false, want: renderer.IconDesktop},
		{name: "darwin with docker flag", osType: model.OSDarwin, isDocker: true, want: renderer.IconApple},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderer.GetOSIcon(tt.osType, tt.isDocker); got != tt.want {
				t.Errorf("GetOSIcon(%v, %v) = %q, want %q", tt.osType, tt.isDocker, got, tt.want)
			}
		})
	}
}
