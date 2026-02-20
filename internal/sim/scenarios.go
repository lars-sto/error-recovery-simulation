package sim

import "time"

func DefaultScenarios(seed int64) []Scenario {
	baseStart := time.Unix(0, 0)
	ids := RTPIDs{MediaSSRC: 1111, FECSSRC: 2222, MediaPT: 96, FECPT: 97}

	baseSender := SenderSpec{
		PacketRateHz:  50,
		PayloadBytes:  1200,
		StartSeq:      1,
		StartTS:       1,
		TimestampStep: 3000,
		StartTime:     baseStart,
	}

	cap2m := NewFloatSchedule(2_000_000)
	bwe2m := NewFloatSchedule(2_000_000)

	mkLink := func(loss LossModel, cap *FloatSchedule) LinkSpec {
		return LinkSpec{
			BaseOneWayDelay: 20 * time.Millisecond,
			Jitter:          5 * time.Millisecond,
			MaxQueueDelay:   200 * time.Millisecond,
			CapacityBps:     cap,
			Loss:            loss,
			Seed:            seed,
		}
	}

	return []Scenario{
		{
			Name:            "bernoulli_2pct",
			Duration:        10 * time.Second,
			IDs:             ids,
			Sender:          baseSender,
			K:               10,
			StaticR:         2,
			StatsInterval:   200 * time.Millisecond,
			BWE:             bwe2m,
			RTTMs:           40,
			JitterMs:        5,
			PlayoutDeadline: 200 * time.Millisecond,
			Link:            mkLink(NewScheduledBernoulliLoss("bernoulli_2pct", seed, NewFloatSchedule(0.02)), cap2m),
			Seed:            seed,
		},
		{
			Name:            "bernoulli_8pct",
			Duration:        10 * time.Second,
			IDs:             ids,
			Sender:          baseSender,
			K:               10,
			StaticR:         2,
			StatsInterval:   200 * time.Millisecond,
			BWE:             bwe2m,
			RTTMs:           40,
			JitterMs:        5,
			PlayoutDeadline: 200 * time.Millisecond,
			Link:            mkLink(NewScheduledBernoulliLoss("bernoulli_8pct", seed, NewFloatSchedule(0.08)), cap2m),
			Seed:            seed,
		},
		{
			Name:            "gilbert_burst",
			Duration:        10 * time.Second,
			IDs:             ids,
			Sender:          baseSender,
			K:               10,
			StaticR:         2,
			StatsInterval:   200 * time.Millisecond,
			BWE:             bwe2m,
			RTTMs:           40,
			JitterMs:        5,
			PlayoutDeadline: 200 * time.Millisecond,
			Link:            mkLink(NewGilbertElliottLoss("gilbert_burst", seed, 0.02, 0.25, 0.002, 0.35), cap2m),
			Seed:            seed,
		},
		{
			Name:            "loss_steps",
			Duration:        12 * time.Second,
			IDs:             ids,
			Sender:          baseSender,
			K:               10,
			StaticR:         2,
			StatsInterval:   200 * time.Millisecond,
			BWE:             bwe2m,
			RTTMs:           40,
			JitterMs:        5,
			PlayoutDeadline: 200 * time.Millisecond,
			Link: mkLink(NewScheduledBernoulliLoss("loss_steps", seed,
				NewFloatSchedule(0.01,
					FloatPoint{At: 0, Value: 0.01},
					FloatPoint{At: 4 * time.Second, Value: 0.08},
					FloatPoint{At: 8 * time.Second, Value: 0.02},
				)), cap2m),
			Seed: seed,
		},
		{
			Name:     "bwe_bottleneck",
			Duration: 12 * time.Second,
			IDs:      ids,
			Sender: SenderSpec{
				PacketRateHz:  120,
				PayloadBytes:  1200,
				StartSeq:      1,
				StartTS:       1,
				TimestampStep: 3000,
				StartTime:     baseStart,
			},
			K:             10,
			StaticR:       2,
			StatsInterval: 200 * time.Millisecond,
			BWE: NewFloatSchedule(2_500_000,
				FloatPoint{At: 0, Value: 2_500_000},
				FloatPoint{At: 4 * time.Second, Value: 1_200_000},
				FloatPoint{At: 8 * time.Second, Value: 2_000_000},
			),
			RTTMs:           40,
			JitterMs:        5,
			PlayoutDeadline: 200 * time.Millisecond,
			Link: mkLink(
				NewScheduledBernoulliLoss("bwe_bottleneck_loss", seed, NewFloatSchedule(0.03)),
				NewFloatSchedule(2_500_000,
					FloatPoint{At: 0, Value: 2_500_000},
					FloatPoint{At: 4 * time.Second, Value: 1_200_000},
					FloatPoint{At: 8 * time.Second, Value: 2_000_000},
				),
			),
			Seed: seed,
		},
	}
}
