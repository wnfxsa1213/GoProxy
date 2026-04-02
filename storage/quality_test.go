package storage

import "testing"

func TestCalculateQualitySnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		proxy         Proxy
		expectedScore int
		expectedGrade string
	}{
		{
			name: "fast and stable becomes s",
			proxy: Proxy{
				Latency:      250,
				SuccessCount: 20,
				FailCount:    0,
			},
			expectedScore: 85,
			expectedGrade: "S",
		},
		{
			name: "fast but failing drops to c",
			proxy: Proxy{
				Latency:      250,
				SuccessCount: 2,
				FailCount:    8,
			},
			expectedScore: 49,
			expectedGrade: "C",
		},
		{
			name: "medium latency without history stays b",
			proxy: Proxy{
				Latency: 800,
			},
			expectedScore: 59,
			expectedGrade: "B",
		},
		{
			name: "risk penalty lowers grade",
			proxy: Proxy{
				Latency:      250,
				SuccessCount: 10,
				FailCount:    0,
				RiskCount:    2,
			},
			expectedScore: 70,
			expectedGrade: "A",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			score, grade := CalculateQualitySnapshot(tc.proxy)
			if score != tc.expectedScore {
				t.Fatalf("score = %d, want %d", score, tc.expectedScore)
			}
			if grade != tc.expectedGrade {
				t.Fatalf("grade = %s, want %s", grade, tc.expectedGrade)
			}
		})
	}
}

func TestCompareProxyQuality(t *testing.T) {
	t.Parallel()

	better := Proxy{QualityScore: 78, Latency: 600, FailCount: 0, SuccessCount: 8}
	worse := Proxy{QualityScore: 62, Latency: 300, FailCount: 0, SuccessCount: 0}
	if CompareProxyQuality(better, worse) <= 0 {
		t.Fatalf("expected better proxy to outrank worse proxy")
	}

	sameScoreLowerLatency := Proxy{QualityScore: 70, Latency: 400}
	sameScoreHigherLatency := Proxy{QualityScore: 70, Latency: 900}
	if CompareProxyQuality(sameScoreLowerLatency, sameScoreHigherLatency) <= 0 {
		t.Fatalf("expected lower latency to win when score ties")
	}
}
