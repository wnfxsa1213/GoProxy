package storage

import "math"

const (
	maxLatencyScore     = 55
	maxSuccessScore     = 20
	maxSampleScore      = 10
	defaultSuccessScore = 10
	defaultSampleScore  = 5
	maxFailPenalty      = 15
	maxRiskPenalty      = 15

	qualityGradeSMin = 85
	qualityGradeAMin = 70
	qualityGradeBMin = 50
)

// CalculateQualityScore 计算代理综合质量分（0-100）
func CalculateQualityScore(p Proxy) int {
	score := latencyScore(p.Latency)
	score += successScore(p.SuccessCount, p.FailCount)
	score += sampleScore(p.SuccessCount + p.FailCount)
	score -= failPenalty(p.FailCount)
	score -= riskPenalty(p.RiskCount)

	switch {
	case score < 0:
		return 0
	case score > 100:
		return 100
	default:
		return score
	}
}

// CalculateQualityGrade 根据综合质量分映射等级
func CalculateQualityGrade(score int) string {
	switch {
	case score >= qualityGradeSMin:
		return "S"
	case score >= qualityGradeAMin:
		return "A"
	case score >= qualityGradeBMin:
		return "B"
	default:
		return "C"
	}
}

// CalculateQualitySnapshot 一次性返回质量分与等级
func CalculateQualitySnapshot(p Proxy) (int, string) {
	score := CalculateQualityScore(p)
	return score, CalculateQualityGrade(score)
}

// CompareProxyQuality 比较两个代理的综合质量
// 返回 1 表示 a 更优，-1 表示 b 更优，0 表示等价
func CompareProxyQuality(a, b Proxy) int {
	switch {
	case a.QualityScore != b.QualityScore:
		if a.QualityScore > b.QualityScore {
			return 1
		}
		return -1
	case a.Latency != b.Latency:
		if a.Latency < b.Latency {
			return 1
		}
		return -1
	case a.FailCount != b.FailCount:
		if a.FailCount < b.FailCount {
			return 1
		}
		return -1
	case a.SuccessCount != b.SuccessCount:
		if a.SuccessCount > b.SuccessCount {
			return 1
		}
		return -1
	default:
		return 0
	}
}

func latencyScore(latencyMs int) int {
	switch {
	case latencyMs <= 0:
		return 0
	case latencyMs <= 300:
		return maxLatencyScore
	case latencyMs <= 500:
		return 50
	case latencyMs <= 800:
		return 44
	case latencyMs <= 1200:
		return 36
	case latencyMs <= 2000:
		return 28
	case latencyMs <= 3000:
		return 16
	case latencyMs <= 4000:
		return 8
	default:
		return 0
	}
}

func successScore(successCount, failCount int) int {
	total := successCount + failCount
	if total <= 0 {
		return defaultSuccessScore
	}
	rate := float64(successCount) / float64(total)
	return int(math.Round(rate * maxSuccessScore))
}

func sampleScore(totalSamples int) int {
	if totalSamples <= 0 {
		return defaultSampleScore
	}
	if totalSamples >= 20 {
		return maxSampleScore
	}
	return totalSamples / 2
}

func failPenalty(failCount int) int {
	penalty := failCount * 3
	if penalty > maxFailPenalty {
		return maxFailPenalty
	}
	return penalty
}

func riskPenalty(riskCount int) int {
	penalty := riskCount * 5
	if penalty > maxRiskPenalty {
		return maxRiskPenalty
	}
	return penalty
}
