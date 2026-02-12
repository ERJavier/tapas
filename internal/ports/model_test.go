package ports

import (
	"testing"
	"time"
)

func TestPort_Project(t *testing.T) {
	tests := []struct {
		dir  string
		want string
	}{
		{"", "—"},
		{"/", "/"},
		{"/tmp", "tmp"},
		{"/home/user/myapp", "myapp"},
	}
	for _, tt := range tests {
		p := Port{WorkingDir: tt.dir}
		if got := p.Project(); got != tt.want {
			t.Errorf("Project(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}

func TestPort_Uptime(t *testing.T) {
	now := time.Now()
	p := Port{StartTime: now.Add(-2 * time.Hour)}
	got := p.Uptime()
	if got < time.Hour || got > 3*time.Hour {
		t.Errorf("Uptime() = %v, expected ~2h", got)
	}
	p0 := Port{}
	if p0.Uptime() != 0 {
		t.Errorf("zero StartTime should give 0 uptime, got %v", p0.Uptime())
	}
}
