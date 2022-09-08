package server

type Config struct {
	network         string
	httpPort        int
	interval        int
	payout          float64
	proxyCount      int
	queueCap        int
	dns             string
	reCaptchaSecret string
	reCaptchaHost   string
}

func NewConfig(network string, httpPort, interval, proxyCount, queueCap int, payout float64, dns, reCaptchaSecret, reCaptchaHost string) *Config {
	return &Config{
		network:         network,
		httpPort:        httpPort,
		interval:        interval,
		payout:          payout,
		proxyCount:      proxyCount,
		queueCap:        queueCap,
		dns:             dns,
		reCaptchaSecret: reCaptchaSecret,
		reCaptchaHost:   reCaptchaHost,
	}
}
