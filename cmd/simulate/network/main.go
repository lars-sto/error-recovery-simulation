package main

import (
	"flag"
	"log"
	"math/rand"
	"time"
)

func main() {
	var (
		seed   = flag.Int64("seed", time.Now().UnixNano(), "rng seed")
		outDir = flag.String("out", "out", "output directory")
	)
	flag.Parse()

	rng := rand.New(rand.NewSource(*seed))

	scenarios := []sim.Scenario{
		sim.RandomLossScenario("random_2pct", 0.02, 20*time.Second, rng.Int63()),
		// sim.BurstGilbertElliottScenario(...): später
	}

	runner := sim.NewRunner(sim.RunnerConfig{
		OutDir: *outDir,
	})

	for _, sc := range scenarios {
		log.Printf("== Scenario: %s ==", sc.Name)
		if err := runner.RunScenario(sc); err != nil {
			log.Fatalf("scenario %s failed: %v", sc.Name, err)
		}
	}
}
