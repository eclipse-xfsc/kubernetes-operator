package openbao

import "testing"

func TestSplitRemoteKey(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantEngine string
		wantPath   string
		wantErr    bool
	}{
		{name: "valid", input: "infrastructure/cassandra/tenant-a", wantEngine: "infrastructure", wantPath: "cassandra/tenant-a"},
		{name: "trim slashes", input: "/resources/postgres/app/", wantEngine: "resources", wantPath: "postgres/app"},
		{name: "missing path", input: "resources", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, secretPath, err := splitRemoteKey(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got engine=%q path=%q", engine, secretPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if engine != tt.wantEngine || secretPath != tt.wantPath {
				t.Fatalf("got engine=%q path=%q, want engine=%q path=%q", engine, secretPath, tt.wantEngine, tt.wantPath)
			}
		})
	}
}
