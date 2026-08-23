package query

import "testing"

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"nanocores_small", "50000000n", "50m"},
		{"nanocores_large", "1500000000n", "1.50"},
		{"nanocores_zero", "0n", "0m"},
		{"millicores_small", "100m", "100m"},
		{"millicores_exact_1000", "1000m", "1.00"},
		{"millicores_large", "2500m", "2.50"},
		{"cores_plain", "2", "2"},
		{"invalid_input", "abc", "abc"},
		{"nanocores_invalid_number", "xyzn", "xyzn"},
		{"millicores_invalid_number", "abcm", "abcm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCPU(tt.input)
			if got != tt.want {
				t.Errorf("FormatCPU(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"ki_small", "1024Ki", "1Mi"},
		{"ki_to_gi", "2097152Ki", "2.00Gi"},
		{"ki_zero", "0Ki", "0Mi"},
		{"mi_small", "128Mi", "128Mi"},
		{"mi_to_gi", "2048Mi", "2.00Gi"},
		{"gi_passthrough", "1Gi", "1Gi"},
		{"gi_large", "4Gi", "4Gi"},
		{"bytes_to_mi", "134217728", "128Mi"},   // 128 * 1024 * 1024
		{"bytes_to_gi", "2147483648", "2.00Gi"}, // 2 * 1024^3
		{"bytes_to_ki", "65536", "64Ki"},        // 64 * 1024, < 1Mi
		{"invalid_input", "abc", "abc"},
		{"ki_invalid_number", "xyzKi", "xyzKi"},
		{"mi_invalid_number", "abcMi", "abcMi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMemory(tt.input)
			if got != tt.want {
				t.Errorf("FormatMemory(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
