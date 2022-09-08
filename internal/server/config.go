package server

type Config struct {
	network         string
	httpPort        int
	interval        int
	payout          int
	proxyCount      int
	queueCap        int
	dns             string
	reCaptchaSecret string
	reCaptchaHost   string
}

func NewConfig(network string, httpPort, interval, payout, proxyCount, queueCap int, dns, reCaptchaSecret, reCaptchaHost string) *Config {
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
