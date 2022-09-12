package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/LK4D4/trylock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/Shib-Chain/shibc-faucet/internal/chain"
)

const (
	AddressKey     string = "address"
	IPKey          string = "ip"
	ReCaptchaToken string = "token"
)

type Server struct {
	chain.TxBuilder
	mutex   trylock.Mutex
	cfg     *Config
	queue   chan Requester
	storage *Storage
}

func NewServer(builder chain.TxBuilder, cfg *Config) *Server {
	return &Server{
		TxBuilder: builder,
		cfg:       cfg,
		queue:     make(chan Requester, cfg.queueCap),
	}
}

func (s *Server) setupRouter() *http.ServeMux {
	router := http.NewServeMux()
	// router.Handle("/", http.FileServer(web.Dist()))
	limiter := NewLimiter(s.cfg.proxyCount, time.Duration(s.cfg.interval)*time.Minute)
	router.Handle("/api/claim", negroni.New(limiter, negroni.Wrap(s.handleClaim())))
	router.Handle("/api/info", s.handleInfo())

	return router
}

func (s *Server) initStorage() {
	log.Info("database: connecting to %s", s.cfg.dns)
	db, err := sqlx.Connect("postgres", s.cfg.dns)
	if err != nil {
		panic(fmt.Errorf("databse: connect db: %v", err))
	}

	err = db.Ping()
	if err != nil {
		panic(fmt.Errorf("datbase: cannot connect: %v", err))
	}
	log.Info("database: Successfully connected!")

	s.migrateDB(db)
	log.Info("database: Migration done!")

	s.storage = NewStorage(db)
}

func (s *Server) migrateDB(db *sqlx.DB) {
	cmdTable := `
		CREATE TABLE IF NOT EXISTS requesters (
			addr VARCHAR(255) PRIMARY KEY,
			ip VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	 	);
	`
	_, err := db.ExecContext(context.Background(), cmdTable)
	if err != nil {
		panic(fmt.Errorf("migration database failed: %w", err))
	}
}

func (s *Server) Run() {
	// init storage
	s.initStorage()

	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			s.consumeQueue()
		}
	}()

	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.UseHandler(s.setupRouter())
	log.Infof("Starting http server %d", s.cfg.httpPort)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(s.cfg.httpPort), n))
}

func (s *Server) consumeQueue() {
	if len(s.queue) == 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for len(s.queue) != 0 {
		requester := <-s.queue
		txHash, err := s.claim(context.Background(), &requester)
		if err != nil {
			log.WithError(err).Error("Failed to handle transaction in the queue")
		} else {
			log.WithFields(log.Fields{
				"txHash":  txHash,
				"address": requester.Addr,
			}).Info("Consume from queue successfully")
		}
	}
}

func (s *Server) handleClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		address := r.PostFormValue(AddressKey)
		// inputIP := r.PostFormValue(IPKey)
		ip := fmt.Sprint(r.Context().Value(IPCtxKey))
		// if inputIP != "" && ip != "::1" && inputIP != ip {
		// 	log.WithFields(log.Fields{
		// 		"addr":     address,
		// 		"ip":       ip,
		// 		"input_ip": inputIP,
		// 	}).Error("Detect different between request's IP and payload's IP")
		// 	errMsg := "System error, please try again later"
		// 	http.Error(w, errMsg, http.StatusServiceUnavailable)
		// 	return
		// }

		if s.cfg.reCaptchaSecret != "" {
			reCaptchaToken := r.PostFormValue(ReCaptchaToken)
			if errMsg, httpStatus := s.verifyReCaptcha(r.Context(), reCaptchaToken); errMsg != "" {
				log.WithFields(log.Fields{
					"addr":  address,
					"ip":    ip,
					"token": reCaptchaToken,
				}).Error(errMsg)
				http.Error(w, errMsg, httpStatus)
				return
			}
		}

		requester, err := s.storage.GetRequester(r.Context(), RequesterFilter{Addr: address, IP: ip})
		if err != nil {
			log.WithFields(log.Fields{
				"addr": address,
				"ip":   ip,
			}).WithError(err).Error("Failed to get requester from storage")
			errMsg := "System error, please try again later"
			http.Error(w, errMsg, http.StatusServiceUnavailable)
			return
		}
		if requester != nil {
			errMsg := fmt.Sprintf("Account:%s already claimed", address)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}

		newReq := Requester{
			Addr: address,
			IP:   ip,
		}

		// Try to lock mutex if the work queue is empty
		// if len(s.queue) != 0 || !s.mutex.TryLock() {
		// 	select {
		// 	case s.queue <- newReq:
		// 		log.WithFields(log.Fields{
		// 			"address": address,
		// 			"ip":      ip,
		// 		}).Info("Added to queue successfully")
		// 		fmt.Fprintf(w, "Added %s to the queue", address)
		// 	default:
		// 		log.Warn("Max queue capacity reached")
		// 		errMsg := "Faucet queue is too long, please try again later"
		// 		http.Error(w, errMsg, http.StatusServiceUnavailable)
		// 	}
		// 	return
		// }

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		txHash, err := s.claim(ctx, &newReq)
		// s.mutex.Unlock()
		if err != nil {
			log.WithFields(log.Fields{
				"addr": address,
				"ip":   ip,
			}).WithError(err).Error("Failed to send transaction")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.WithFields(log.Fields{
			"txHash":  txHash,
			"address": address,
		}).Info("Funded directly successfully")

		json.NewEncoder(w).Encode(struct {
			TxHash string `json:"tx_hash"`
		}{
			TxHash: txHash.String(),
		})
	}
}

func (s *Server) verifyReCaptcha(ctx context.Context, reCaptchaToken string) (errMsg string, httpStatusCode int) {
	const (
		verifyCaptchaGoogleAPI = "https://www.google.com/recaptcha/api/siteverify"
		systemErrMsg           = "System error, please try again later"
	)
	captchaPayloadRequest := url.Values{}
	captchaPayloadRequest.Set("secret", s.cfg.reCaptchaSecret)
	captchaPayloadRequest.Set("response", reCaptchaToken)

	verifyCaptchaRequest, err := http.NewRequest(http.MethodPost, verifyCaptchaGoogleAPI, strings.NewReader(captchaPayloadRequest.Encode()))
	if err != nil {
		log.WithError(err).Error("Init requeset verify captcha failed")
		return systemErrMsg, http.StatusServiceUnavailable
	}
	verifyCaptchaRequest.Header.Add("content-type", "application/x-www-form-urlencoded")
	verifyCaptchaRequest.Header.Add("cache-control", "no-cache")

	verifyCaptchaResponse, err := http.DefaultClient.Do(verifyCaptchaRequest)
	if err != nil {
		log.WithError(err).Error("Failed to verify reCaptcha token")
		return systemErrMsg, http.StatusServiceUnavailable
	}

	captchaVerifyResponse := struct {
		Success  bool   `json:"success"`
		HostName string `json:"hostname"`
	}{}
	if err = json.NewDecoder(verifyCaptchaResponse.Body).Decode(&captchaVerifyResponse); err != nil {
		log.WithError(err).Error("Failed to decode reCaptcha response")
		return systemErrMsg, http.StatusServiceUnavailable
	}
	defer verifyCaptchaResponse.Body.Close()

	log.WithField("reCaptcha_response", captchaVerifyResponse).Info("verify reCaptcha response")
	if !captchaVerifyResponse.Success || captchaVerifyResponse.HostName != s.cfg.reCaptchaHost {
		return "Invalid reCAPTCHA. Please try again.", http.StatusBadRequest
	}

	return "", 0
}

func (s *Server) claim(ctx context.Context, requester *Requester) (common.Hash, error) {
	txHash, err := s.Transfer(context.Background(), requester.Addr, chain.EtherToWei(s.cfg.payout))
	if err != nil {
		return common.Hash{}, err
	}

	requester.CreatedAt = time.Now()
	if err := s.storage.CreateRequester(context.Background(), requester); err != nil {
		log.WithFields(log.Fields{
			"txHash":  txHash,
			"address": requester.Addr,
			"ip":      requester.IP,
		}).WithError(err).Error("Cannot store requester")
	}

	return txHash, nil
}

func (s *Server) handleInfo() http.HandlerFunc {
	type info struct {
		Account string  `json:"account"`
		Network string  `json:"network"`
		Payout  float64 `json:"payout"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info{
			Account: s.Sender().String(),
			Network: s.cfg.network,
			Payout:  s.cfg.payout,
		})
	}
}
