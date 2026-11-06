package telemetry

import "testing"

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: "valid", config: Config{Protocol: "http/protobuf", TracesSampler: "parentbased_always_on"}},
		{name: "unsupported protocol", config: Config{Protocol: "grpc", TracesSampler: "parentbased_always_on"}, wantErr: true},
		{name: "unsupported sampler", config: Config{Protocol: "http/protobuf", TracesSampler: "random"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
