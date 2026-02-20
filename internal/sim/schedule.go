package sim

import (
	"sort"
)

func NewFloatSchedule(defaultVal float64, points ...FloatPoint) *FloatSchedule {
	p := append([]FloatPoint(nil), points...)
	sort.Slice(p, func(i, j int) bool { return p[i].At < p[j].At })
	return &FloatSchedule{Points: p, Default: defaultVal}
}
