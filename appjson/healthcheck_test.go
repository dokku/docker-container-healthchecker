package appjson

import "testing"

func TestHealthcheck_validateAddresses(t *testing.T) {
	type fields struct {
		Port int
	}
	tests := []struct {
		name      string
		fields    fields
		addresses map[string]bool
		wantErr   bool
	}{
		{
			name:      "when addresses are empty",
			fields:    fields{},
			addresses: map[string]bool{},
			wantErr:   false,
		},
		{
			name:      "when addresses are not empty",
			fields:    fields{Port: 5000},
			addresses: map[string]bool{":::5000": true},
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Healthcheck{
				Port: tt.fields.Port,
			}
			err := h.validateAddresses(tt.addresses)
			if (err != nil) != tt.wantErr {
				t.Errorf("Healthcheck.determineErrorFor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
