package sim

func (e *Env) EmitSummary() {
	e.log.Summary("mode", string(e.mode))
	e.log.Summary("scenario", e.sc.Name)
	e.log.Summary("seed", e.sc.Seed)

	e.log.Summary("sent_media", e.sentMedia.Load())
	e.log.Summary("sent_fec", e.sentFEC.Load())
	e.log.Summary("recv_media", e.recvMedia.Load())
	e.log.Summary("recv_fec", e.recvFEC.Load())

	// simple derived metrics (placeholder)
	sent := float64(e.sentMedia.Load())
	recv := float64(e.recvMedia.Load())
	if sent > 0 {
		e.log.Summary("media_delivery_ratio", recv/sent)
	}
}
