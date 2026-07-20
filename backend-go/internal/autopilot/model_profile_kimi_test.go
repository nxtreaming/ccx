package autopilot

import "testing"

func TestInferModelFamily_KimiCodeModels(t *testing.T) {
	for _, model := range []string{"k3", "k3[1m]", "kimi-for-coding", "kimi-for-coding-highspeed"} {
		if got := InferModelFamily(model, ""); got != ModelFamilyKimi {
			t.Errorf("InferModelFamily(%q) = %q, want %q", model, got, ModelFamilyKimi)
		}
	}
}

func TestModelProfileQualityTierFromFamily_KimiCodeModels(t *testing.T) {
	tests := []struct {
		model string
		want  QualityTier
	}{
		{model: "k3", want: QualityTierPremium},
		{model: "k3[1m]", want: QualityTierPremium},
		{model: "kimi-for-coding", want: QualityTierHigh},
		{model: "kimi-for-coding-highspeed", want: QualityTierHigh},
		{model: "kimi-k2.6", want: QualityTierHigh},
	}
	for _, tt := range tests {
		if got := ModelProfileQualityTierFromFamily(ModelFamilyKimi, tt.model); got != tt.want {
			t.Errorf("ModelProfileQualityTierFromFamily(%q) = %q, want %q", tt.model, got, tt.want)
		}
	}
}
